package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// ==================== 原有 ReAct 回调（保留兼容） ====================

type AgentCallback struct {
	startTime   time.Time
	stepCount   int
	messages    map[int]*schema.Message // 完整对话历史：User + Assistant + Tool
	pendingTool *schema.Message
	OnReasoning func(msg *schema.Message)
}

func (a *AgentCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	a.startTime = time.Now()
	a.stepCount++
	log.WithCtx(ctx).Debugw("OnStart", "component", info.Component, "type", fmt.Sprintf("%T", input), "input", input, "step", a.stepCount)
	return ctx
}

func (a *AgentCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	duration := time.Since(a.startTime)
	log.WithCtx(ctx).Debugw("OnEnd", "component", info.Component,
		"duration", duration, "step", a.stepCount,
		"type", fmt.Sprintf("%T", output),
		"output", output)
	return ctx
}

func (a *AgentCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	log.WithCtx(ctx).Errorw("OnError",
		"component", info.Component,
		"step", a.stepCount,
		"error", err)
	return ctx
}

func (a *AgentCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	log.WithCtx(ctx).Debugw("OnStartWithStreamInput", "component", info.Component,
		"input", input, "type", fmt.Sprintf("%T", input), "step", a.stepCount)
	return ctx
}

func (a *AgentCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	graphInfoName := "PlanExecuteAgent" // 更新为新的 Agent 名称

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close()

		fmt.Println("=========[OnEndStream]=========")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			if info.Name == graphInfoName {
				if msg, ok := frame.(*schema.Message); ok {
					if len(msg.ReasoningContent) > 0 {
						a.OnReasoning(msg)
					}
				}
				fmt.Printf("%s: %s", info.Name, string(s))
			}
		}
	}()
	return ctx
}

// ==================== Plan-Execute 模式回调 ====================

// PlanExecuteCallback Plan-Execute 模式的回调处理器
type PlanExecuteCallback struct {
	mu             sync.Mutex
	startTime      time.Time
	currentPlan    *Plan
	executedSteps  []PlanStep
	messageHandler func(ctx context.Context, stage string, data map[string]any)
}

// NewPlanExecuteCallback 创建 Plan-Execute 回调
func NewPlanExecuteCallback(handler func(ctx context.Context, stage string, data map[string]any)) *PlanExecuteCallback {
	return &PlanExecuteCallback{
		messageHandler: handler,
		executedSteps:  make([]PlanStep, 0),
	}
}

// OnPlanningStart 规划开始
func (c *PlanExecuteCallback) OnPlanningStart(ctx context.Context, query string) {
	c.startTime = time.Now()
	c.emit(ctx, "planning_start", map[string]any{
		"query":     query,
		"timestamp": time.Now().Unix(),
	})
}

// OnPlanningComplete 规划完成
func (c *PlanExecuteCallback) OnPlanningComplete(ctx context.Context, plan *Plan) {
	c.mu.Lock()
	c.currentPlan = plan
	c.mu.Unlock()

	planJSON, _ := json.Marshal(plan)
	c.emit(ctx, "planning_complete", map[string]any{
		"plan":       string(planJSON),
		"step_count": len(plan.Steps),
		"duration":   time.Since(c.startTime).String(),
	})
}

// OnStepStart 步骤开始
func (c *PlanExecuteCallback) OnStepStart(ctx context.Context, step *PlanStep, index int) {
	c.emit(ctx, "step_start", map[string]any{
		"step_id":     step.ID,
		"index":       index,
		"description": step.Description,
		"tool_name":   step.ToolName,
		"tool_args":   step.ToolArgs,
		"reason":      step.Reason,
	})
}

// OnStepComplete 步骤完成
func (c *PlanExecuteCallback) OnStepComplete(ctx context.Context, step *PlanStep, result string, index int) {
	c.mu.Lock()
	c.executedSteps = append(c.executedSteps, *step)
	c.mu.Unlock()

	c.emit(ctx, "step_complete", map[string]any{
		"step_id": step.ID,
		"index":   index,
		"status":  step.Status,
		"result":  result,
	})
}

// OnStepError 步骤错误
func (c *PlanExecuteCallback) OnStepError(ctx context.Context, step *PlanStep, err error, index int) {
	c.emit(ctx, "step_error", map[string]any{
		"step_id": step.ID,
		"index":   index,
		"error":   err.Error(),
	})
}

// OnReplanStart 重规划开始
func (c *PlanExecuteCallback) OnReplanStart(ctx context.Context, reason string) {
	c.emit(ctx, "replan_start", map[string]any{
		"reason": reason,
	})
}

// OnReplanComplete 重规划完成
func (c *PlanExecuteCallback) OnReplanComplete(ctx context.Context, newPlan *Plan) {
	c.mu.Lock()
	c.currentPlan = newPlan
	c.mu.Unlock()

	planJSON, _ := json.Marshal(newPlan)
	c.emit(ctx, "replan_complete", map[string]any{
		"new_plan": string(planJSON),
	})
}

// OnExecutionComplete 执行完成
func (c *PlanExecuteCallback) OnExecutionComplete(ctx context.Context, answer string) {
	c.emit(ctx, "execution_complete", map[string]any{
		"answer":     answer,
		"total_time": time.Since(c.startTime).String(),
		"step_count": len(c.executedSteps),
	})
}

// OnFinalAnswerStream 最终答案流式输出
func (c *PlanExecuteCallback) OnFinalAnswerStream(ctx context.Context, chunk string) {
	c.emit(ctx, "answer_stream", map[string]any{
		"chunk": chunk,
	})
}

// emit 发送事件
func (c *PlanExecuteCallback) emit(ctx context.Context, stage string, data map[string]any) {
	if c.messageHandler != nil {
		c.messageHandler(ctx, stage, data)
	}
	log.WithCtx(ctx).Debugw("PlanExecuteCallback", "stage", stage, "data", data)
}

// GetCurrentPlan 获取当前计划
func (c *PlanExecuteCallback) GetCurrentPlan() *Plan {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentPlan
}

// GetExecutedSteps 获取已执行步骤
func (c *PlanExecuteCallback) GetExecutedSteps() []PlanStep {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.executedSteps
}

// ==================== 消息处理（保留原有功能） ====================

type MessageFuture struct {
	MsgCallback     func(ctx context.Context, stage string, m map[string]any)
	seenToolCalls   map[string]bool
	seenToolResults map[string]bool
	seenMessages    map[string]bool
	seenSetup       int
	mu              sync.RWMutex
}

func NewMessageFuture() *MessageFuture {
	return &MessageFuture{
		seenToolCalls:   make(map[string]bool),
		seenToolResults: make(map[string]bool),
		seenMessages:    make(map[string]bool),
	}
}

func (m *MessageFuture) ResetDedup() {
	m.mu.Lock()
	clear(m.seenToolCalls)
	clear(m.seenToolResults)
	clear(m.seenMessages)
	m.seenSetup = 0
	m.mu.Unlock()
}

// WithMessageFuture 返回 ReAct Agent 所需的选项和消息未来处理器
func (m *MessageFuture) WithMessageFuture() (agent.AgentOption, react.MessageFuture) {
	return react.WithMessageFuture()
}

// ProcessMessageFuture 处理 ReAct Agent 的中间消息流
func (m *MessageFuture) ProcessMessageFuture(ctx context.Context, msgFuture react.MessageFuture) {
	iter := msgFuture.GetMessageStreams()
	for {
		msg, hasNext, err := iter.Next()
		if err != nil {
			log.WithCtx(ctx).Error("Error reading stream: %v", err)
			break
		}
		if !hasNext {
			log.WithCtx(ctx).Warn("All messages processed")
			break
		}

		ss, err := schema.ConcatMessageStream(msg)
		if err != nil {
			log.WithCtx(ctx).Error("Error concat message stream: %v", err)
			continue
		}

		if ss != nil {
			// 注意：HandleIntermediateMessage 内部会调用 MsgCallback
			// 现在 MsgCallback 是发送到 channel，所以这里是安全的
			m.HandleIntermediateMessage(ctx, ss)
		}
	}
}

// HandleIntermediateMessage 处理中间消息（用于流式输出）
func (m *MessageFuture) HandleIntermediateMessage(ctx context.Context, msg *schema.Message) {
	switch {
	case len(msg.ToolCalls) > 0:
		var first int
		for _, call := range msg.ToolCalls {
			if call.ID == "" {
				continue
			}
			m.mu.Lock()
			key := call.ID
			if m.seenToolCalls[key] {
				m.mu.Unlock()
				continue
			}

			m.seenToolCalls[key] = true
			m.mu.Unlock()
			m.seenSetup++
			if first == 0 {
				m.MsgCallback(ctx, "tool_call_reasoning_content", map[string]any{
					"event":             "tool_call",
					"reasoning_content": msg.ReasoningContent,
				})
				m.MsgCallback(ctx, "tool_call", map[string]any{
					"setup":    m.seenSetup,
					"event":    "tool_call",
					"toolCall": call,
				})
				first++
			} else {
				m.MsgCallback(ctx, "tool_call", map[string]any{
					"setup":    m.seenSetup,
					"event":    "tool_call",
					"toolCall": call,
				})
			}
		}
	case msg.Role == schema.Tool:
		m.seenSetup++

		if msg.ToolCallID == "" {
			m.MsgCallback(ctx, "tool_result", map[string]any{"msg": msg})
			return
		}
		m.mu.Lock()
		if m.seenToolResults[msg.ToolCallID] {
			m.mu.Unlock()
			return
		}
		m.seenToolResults[msg.ToolCallID] = true
		m.mu.Unlock()

		m.MsgCallback(ctx, "tool_result", map[string]any{
			"setup":      m.seenSetup,
			"event":      "tool_result",
			"toolCallID": msg.ToolCallID,
			"content":    msg.Content,
		})

	case msg.Role == schema.Assistant && (msg.Content != "" || msg.ReasoningContent != ""):
		m.seenSetup++
		_, ok := m.seenMessages[msg.ReasoningContent]
		if ok {
			return
		}
		m.seenMessages[msg.ReasoningContent] = true
		m.MsgCallback(ctx, "thinking", map[string]any{
			"setup":             m.seenSetup,
			"event":             "thinking",
			"reasoning_content": msg.ReasoningContent,
		})

	default:
		log.WithCtx(ctx).Debug("Unknown message type", msg)
	}
}
