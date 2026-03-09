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
	//fmt.Println("==================")
	//inputStr, _ := json.MarshalIndent(input, "", "  ")
	//fmt.Printf("[OnStart] %s\n", string(inputStr))
	return ctx
}

func (a *AgentCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	duration := time.Since(a.startTime)
	// default:
	log.WithCtx(ctx).Debugw("OnEnd", "component", info.Component,
		"duration", duration, "step", a.stepCount,
		"type", fmt.Sprintf("%T", output),
		"output", output)
	//fmt.Println("=========[OnEnd]=========")
	//outputStr, _ := json.MarshalIndent(output, "", "  ")
	//fmt.Println(string(outputStr))
	return ctx
}

func (a *AgentCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	// 识别限流错误
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

	//fmt.Println("=========[OnStartWithStreamInput]=========")
	//inputStr, _ := json.MarshalIndent(input, "", "  ")
	//fmt.Println(string(inputStr))
	return ctx
}

func (a *AgentCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	//log.WithCtx(ctx).Debugw("OnEndWithStreamOutput", "component", info.Component,
	//	"output", output, "type", fmt.Sprintf("%T", output), "step", a.stepCount)
	var graphInfoName = react.GraphName

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close() // remember to close the stream in defer

		fmt.Println("=========[OnEndStream]=========")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				// finish
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

			if info.Name == graphInfoName { // 仅打印 graph 的输出, 否则每个 stream 节点的输出都会打印一遍

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

type MessageFuture struct {
	MsgCallback     func(ctx context.Context, stage string, m map[string]any)
	seenToolCalls   map[string]bool // key: toolCall.ID
	seenToolResults map[string]bool // key: toolCallID
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

func (m *MessageFuture) WithMessageFuture() (agent.AgentOption, react.MessageFuture) {
	return react.WithMessageFuture()
}

// HandleIntermediateMessage 处理中间消息
func (m *MessageFuture) HandleIntermediateMessage(ctx context.Context, msg *schema.Message) {
	switch {
	case len(msg.ToolCalls) > 0:
		var first int
		for _, call := range msg.ToolCalls {
			if call.ID == "" {
				continue // 兜底
			}
			m.mu.Lock()
			key := call.ID
			if m.seenToolCalls[key] {
				m.mu.Unlock()
				continue // 已发送，跳过
			}

			m.seenToolCalls[key] = true
			m.mu.Unlock()
			m.seenSetup++
			if first == 0 {
				// 你的原有逻辑（排序等）
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
				// 你的原有逻辑（排序等）
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
		// thinking 消息通常不需要严格去重（内容会变化），直接发
		_, ok := m.seenMessages[msg.ReasoningContent]
		if ok {
			return
		}
		m.seenMessages[msg.ReasoningContent] = true
		m.MsgCallback(ctx, "thinking", map[string]any{
			"setup":             m.seenSetup,
			"event":             "thinking",
			"reasoning_content": msg.ReasoningContent,
			//"content":          msg, // 或只发 Content + ReasoningContent
		})

	default:
		log.WithCtx(ctx).Debug("Unknown message type  ", msg)
	}
}

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
