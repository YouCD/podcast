package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/youcd/toolkit/log"

	"podcast/internal/ai/agent"
	"podcast/internal/database/models"
)

// ChatInput 请求结构体
type ChatInput struct {
	Content   string `json:"content" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
}

// ChatResponse 响应结构体
type ChatResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// SSEEvent 定义SSE事件结构
type SSEEvent struct {
	Event string
	Data  string
}

func (h *ChatHandler) StreamChat(c *gin.Context) {
	var input ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		ErrorWithMessage(c, "Invalid request format")
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("X-Accel-Buffering", "no") // 禁用Nginx缓冲
	c.Header("Cache-Control", "no-cache, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", "x-requested-with,content-type,authorization")

	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	// 加载历史消息
	historyModels, err := h.chatService.GetMessagesBySessionID(input.SessionID)
	if err != nil {
		log.WithCtx(ctx).Errorf("Failed to load history: %v", err)
		sendSSEError(c.Writer, flusher, "Failed to load history")
		return
	}

	//sort.Slice(historyModels, func(i, j int) bool {
	//	return historyModels[i].CreatedAt.Before(historyModels[j].CreatedAt)
	//})
	if len(historyModels) == 0 {
		historyModels = []*models.Message{}
	}
	if len(historyModels) > 10 {
		historyModels = historyModels[len(historyModels)-10:]
	}

	// 转换为 schema.Message 列表
	var messages []*schema.Message
	for _, m := range historyModels {
		role := schema.User
		if m.Role == "assistant" {
			role = schema.Assistant
		} else if m.Role == "system" {
			role = schema.System
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	log.WithCtx(ctx).Infow("RagAgent", "messages", messages)

	// 保存用户消息（保证多轮历史可被后续请求加载）
	userID := 0
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(int); ok {
			userID = id
		}
	}
	session, err := h.chatService.GetSessionBySessionID(ctx, input.SessionID)
	if err != nil {
		session = &models.ChatSession{SessionID: input.SessionID, UserID: userID, Title: input.Content}
		if err := h.chatService.CreateSession(ctx, session); err != nil {
			log.WithCtx(ctx).Errorf("Failed to create session: %v", err)
		}
	}
	userMsg := &models.Message{
		SessionID: input.SessionID,
		UUID:      uuid.NewString(),
		Role:      "user",
		Content:   input.Content,
	}
	if err := h.chatService.CreateMessage(userMsg); err != nil {
		log.WithCtx(ctx).Errorf("Failed to save user message: %v", err)
	}
	session.UpdatedAt = userMsg.CreatedAt
	h.chatService.UpdateSession(ctx, session)

	// 创建一个带缓冲的 Channel 用于串行化 SSE 事件
	// 缓冲大小可以根据实际情况调整，防止阻塞生产者
	sseChan := make(chan SSEEvent, 100)

	// 使用 WaitGroup 等待所有生产者 goroutine 结束
	var wg sync.WaitGroup

	// 启动 Goroutine 运行 Agent 并处理主输出流
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(sseChan) // 生产者结束时关闭 Channel

		// 获取共享 Agent 实例（首次调用时初始化，之后复用）
		agentInstance, err := h.getAgent(ctx)
		if err != nil {
			log.WithCtx(ctx).Errorf("Failed to create RAG Agent: %v", err)
			errResp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("Failed to create RAG Agent: %v", err)})
			sseChan <- SSEEvent{Event: "error", Data: string(errResp)}
			return
		}

		// 设置消息回调处理函数（同时捕获计划数据，供历史渲染）
		var capturedPlanJSON string
		messageHandler := func(ctx context.Context, stage string, data map[string]any) {
			log.WithCtx(ctx).Debugw(stage, "data", data)
			switch stage {
			case "plan_created":
				if p, ok := data["plan"].(string); ok {
					capturedPlanJSON = p
				}
			case "replanned":
				// 仅当新计划有步骤时才替换（避免空计划覆盖原计划）
				if b, err := json.Marshal(data["plan"]); err == nil {
					var np struct {
						Steps []json.RawMessage `json:"steps"`
					}
					if json.Unmarshal(b, &np) == nil && len(np.Steps) > 0 {
						capturedPlanJSON = string(b)
					}
				}
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.WithCtx(ctx).Errorf("Failed to marshal response: %v", err)
				return
			}
			select {
			case sseChan <- SSEEvent{Event: stage, Data: string(jsonData)}:
			case <-ctx.Done():
			}
		}

		// 构建 Plan-Execute Agent
		planExecuteAgent, err := agentInstance.BuildPlanExecuteAgent(messageHandler)
		if err != nil {
			log.WithCtx(ctx).Errorf("Failed to init Plan-Execute agent: %v", err)
			errResp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("Failed to init Plan-Execute agent: %v", err)})
			sseChan <- SSEEvent{Event: "error", Data: string(errResp)}
			return
		}

		// 构建用户输入消息
		userMessage := schema.UserMessage(input.Content)

		// 启动 Plan-Execute Agent 流式处理（携带多轮对话历史）
		stream, err := planExecuteAgent.Stream(ctx, userMessage, messages)
		if err != nil {
			log.WithCtx(ctx).Errorf("RAG query failed: %v", err)
			errResp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("RAG query failed: %v", err)})
			sseChan <- SSEEvent{Event: "error", Data: string(errResp)}
			return
		}

		// 累积助手回复，流结束后保存到会话历史
		var assistantContent, thinkContent strings.Builder
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						sseChan <- SSEEvent{Event: "end", Data: `{"end":true}`}
						if assistantContent.Len() > 0 {
							planJSON, stepsJSON := h.buildPlanAndStepsJSON(planExecuteAgent, capturedPlanJSON)
							h.saveAssistantMessage(input.SessionID, assistantContent.String(), thinkContent.String(), planJSON, stepsJSON)
						}
						return
					}
					log.WithCtx(ctx).Errorf("Error reading stream: %v", err)
					return
				}
				if msg.ReasoningContent != "" {
					thinkContent.WriteString(msg.ReasoningContent)
				}
				if msg.Content != "" {
					assistantContent.WriteString(msg.Content)
				}
				handlerStreamMsg(msg, sseChan)
			}
		}
	}()

	// 4. 主线程：唯一的写入者，从 Channel 读取并写入 HTTP Response
	// 启动一个 goroutine 来等待所有生产者完成，并关闭 channel
	go func() {
		wg.Wait()
	}()

	// 消费 Channel 中的事件
	for event := range sseChan {
		_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Event, event.Data)
		flusher.Flush()

		// 检查连接状态
		if ctx.Err() != nil {
			return
		}
	}
}

// buildPlanAndStepsJSON 生成历史渲染所需的 plan/steps JSON
// plan 优先用事件流捕获的计划（含 tool_args），steps 用实际执行过的步骤（含状态与结果）
func (h *ChatHandler) buildPlanAndStepsJSON(a *agent.PlanExecuteAgent, capturedPlanJSON string) (string, string) {
	planJSON := capturedPlanJSON
	if planJSON == "" {
		if plan := a.GetCurrentPlan(); plan != nil && len(plan.Steps) > 0 {
			if b, err := json.Marshal(plan); err == nil {
				planJSON = string(b)
			}
		}
	}
	if planJSON == "" {
		return "", ""
	}
	// steps 数组格式与前端 StepInfo 对齐（step_id 而非 id）
	steps := make([]map[string]any, 0)
	for _, s := range a.GetExecutedSteps() {
		step := map[string]any{
			"step_id":     s.ID,
			"description": s.Description,
			"reason":      s.Reason,
			"tool_name":   s.ToolName,
			"status":      s.Status,
		}
		if s.Result != "" {
			step["result"] = s.Result
		}
		steps = append(steps, step)
	}
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return planJSON, ""
	}
	return planJSON, string(stepsJSON)
}

// saveAssistantMessage 保存助手回复到会话历史（含思考过程、计划与步骤，供刷新后渲染）
func (h *ChatHandler) saveAssistantMessage(sessionID, content, thinkContent, planJSON, stepsJSON string) {
	ctx := context.Background()
	msg := &models.Message{
		SessionID:        sessionID,
		UUID:             uuid.NewString(),
		Role:             "assistant",
		Content:          content,
		ReasoningContent: thinkContent,
		ThinkContent:     thinkContent,
		Plan:             planJSON,
		Steps:            stepsJSON,
	}
	if err := h.chatService.CreateMessage(msg); err != nil {
		log.WithCtx(ctx).Errorf("Failed to save assistant message: %v", err)
		return
	}
	if session, err := h.chatService.GetSessionBySessionID(ctx, sessionID); err == nil {
		session.UpdatedAt = msg.CreatedAt
		h.chatService.UpdateSession(ctx, session)
	}
}

func handlerStreamMsg(msg *schema.Message, sseChan chan SSEEvent) {
	if msg.ReasoningContent != "" {
		response := ChatResponse{Message: msg.ReasoningContent}
		if jsonData, err := json.Marshal(response); err == nil {
			sseChan <- SSEEvent{Event: "think", Data: string(jsonData)}
		}
	}
	if msg.Content != "" {
		response := ChatResponse{Message: msg.Content}
		if jsonData, err := json.Marshal(response); err == nil {
			sseChan <- SSEEvent{Event: "message", Data: string(jsonData)}
		}
	}
}

// 辅助函数：发送SSE错误
func sendSSEError(w io.Writer, flusher http.Flusher, message string) {
	errorResp := map[string]string{
		"error": message,
	}
	jsonData, _ := json.Marshal(errorResp)
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", jsonData)
	if flusher != nil {
		flusher.Flush()
	}
}
