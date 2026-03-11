package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"podcast/internal/ai/agent"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/youcd/toolkit/log"

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

	// 创建一个带缓冲的 Channel 用于串行化 SSE 事件
	// 缓冲大小可以根据实际情况调整，防止阻塞生产者
	sseChan := make(chan SSEEvent, 100)

	// 使用 WaitGroup 等待所有生产者 goroutine 结束
	var wg sync.WaitGroup

	future := agent.NewMessageFuture()
	defer future.ResetDedup()

	// 1. 设置回调：将事件发送到 Channel 而不是直接写入
	future.MsgCallback = func(ctx context.Context, stage string, m map[string]any) {
		log.WithCtx(ctx).Debugw(stage, "msg", m)
		//jsonData, err := json.Marshal(m)
		//if err != nil {
		//	log.WithCtx(ctx).Errorf("Failed to marshal response: %v", err)
		//	return
		//}
		//
		//select {
		//case sseChan <- SSEEvent{Event: stage, Data: string(jsonData)}:
		//case <-ctx.Done():
		//	// 上下文取消，丢弃消息
		//}
	}

	futureOpt, msgFuture := future.WithMessageFuture()

	// 2. 启动 Goroutine 处理中间消息
	wg.Add(1)
	go func() {
		defer wg.Done()
		future.ProcessMessageFuture(ctx, msgFuture)
	}()

	// 3. 启动 Goroutine 运行 Agent 并处理主输出流
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(sseChan) // 生产者结束时关闭 Channel

		// 创建 Agent 实例
		agentInstance, err := agent.NewRAGAgent(ctx, h.ragCfg)
		if err != nil {
			log.WithCtx(ctx).Errorf("Failed to create RAG Agent: %v", err)
			errResp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("Failed to create RAG Agent: %v", err)})
			sseChan <- SSEEvent{Event: "error", Data: string(errResp)}
			return
		}

		// 启动 Agent 流式处理
		stream, err := agentInstance.Stream(ctx, messages, futureOpt)
		if err != nil {
			log.WithCtx(ctx).Errorf("RAG query failed: %v", err)
			errResp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("RAG query failed: %v", err)})
			sseChan <- SSEEvent{Event: "error", Data: string(errResp)}
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						sseChan <- SSEEvent{Event: "end", Data: `{"end":true}`}
						return
					}
					log.WithCtx(ctx).Errorf("Error reading stream: %v", err)
					return
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
