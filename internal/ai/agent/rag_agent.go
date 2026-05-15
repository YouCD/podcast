package agent

import (
	"context"
	"errors"
	"io"

	"podcast/internal/ai/llm"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// AgentMode Agent 运行模式
type AgentMode string

const (
	// ModeReAct ReAct 循环模式
	ModeReAct AgentMode = "react"
	// ModePlanExecute Plan-and-Execute 模式
	ModePlanExecute AgentMode = "plan_execute"
)

// RAGAgentConfig RAG Agent 配置
type RAGAgentConfig struct {
	MCP       *MCPConfig
	RagConfig *types.RagConfig
	LLMPool   *llm.LLMPool
	// Agent 模式，默认为 Plan-Execute
	Mode AgentMode
	// 最大重规划次数（仅 Plan-Execute 模式有效）
	MaxReplan int
}

// RAGAgent RAG Agent
type RAGAgent struct {
	chatModel model.ToolCallingChatModel
	tools     []tool.BaseTool
	mode      AgentMode
	maxReplan int
}

// NewAgent 创建 Agent（默认使用 Plan-Execute 模式）
func NewAgent(ctx context.Context, cfg *RAGAgentConfig) (*RAGAgent, error) {
	// 初始化 MCP 工具
	tm, err := NewToolManager(ctx, cfg.MCP, cfg.RagConfig)
	if err != nil {
		return nil, err
	}

	// 创建 ChatModel
	m, err := NewRetryChatModel(ctx, cfg.LLMPool, 5, defaultBackoff, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		log.WithCtx(ctx).Error("Failed to init retry model", err)
		return nil, err
	}

	// 设置默认模式
	mode := cfg.Mode
	if mode == "" {
		mode = ModePlanExecute
	}

	return &RAGAgent{
		chatModel: m,
		tools:     tm.GetTools(),
		mode:      mode,
		maxReplan: cfg.MaxReplan,
	}, nil
}

// BuildRAGAgent 构建 ReAct Agent（保留原有接口兼容）
func (r *RAGAgent) BuildRAGAgent(ctx context.Context) (*react.Agent, error) {
	// 使用Eino内置的ReAct Agent
	config := &react.AgentConfig{
		ToolCallingModel: r.chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: r.tools},
		// 最大迭代次数，防止无限循环
		MaxStep:               20,
		StreamToolCallChecker: toolCallChecker,
	}
	return react.NewAgent(ctx, config)
}

// BuildPlanExecuteAgent 构建 Plan-Execute Agent
func (r *RAGAgent) BuildPlanExecuteAgent(messageHandler func(ctx context.Context, stage string, data map[string]any)) (*PlanExecuteAgent, error) {
	config := &PlanExecuteAgentConfig{
		ChatModel:     r.chatModel,
		Tools:         r.tools,
		MaxReplan:     r.maxReplan,
		MessageHandle: messageHandler,
	}
	return NewPlanExecuteAgent(config)
}

// BuildPlanExecuteAgentWithReplan 构建支持重规划的 Plan-Execute Agent
func (r *RAGAgent) BuildPlanExecuteAgentWithReplan(messageHandler func(ctx context.Context, stage string, data map[string]any), maxReplan int) (*PlanExecuteAgentWithReplan, error) {
	config := &PlanExecuteAgentConfig{
		ChatModel:     r.chatModel,
		Tools:         r.tools,
		MaxReplan:     maxReplan,
		MessageHandle: messageHandler,
	}
	return NewPlanExecuteAgentWithReplan(config, maxReplan)
}

// GetMode 获取当前 Agent 模式
func (r *RAGAgent) GetMode() AgentMode {
	return r.mode
}

// SetMode 设置 Agent 模式
func (r *RAGAgent) SetMode(mode AgentMode) {
	r.mode = mode
}

// GetTools 获取工具列表
func (r *RAGAgent) GetTools() []tool.BaseTool {
	return r.tools
}

// GetChatModel 获取 ChatModel
func (r *RAGAgent) GetChatModel() model.ToolCallingChatModel {
	return r.chatModel
}

// Run 执行 Agent（根据模式自动选择）
func (r *RAGAgent) Run(ctx context.Context, input *schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.Message, error) {
	switch r.mode {
	case ModeReAct:
		return r.runReAct(ctx, input)
	case ModePlanExecute:
		return r.runPlanExecute(ctx, input, messageHandler)
	default:
		return r.runPlanExecute(ctx, input, messageHandler)
	}
}

// Stream 流式执行 Agent
func (r *RAGAgent) Stream(ctx context.Context, input *schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.StreamReader[*schema.Message], error) {
	switch r.mode {
	case ModeReAct:
		return r.streamReAct(ctx, input)
	case ModePlanExecute:
		return r.streamPlanExecute(ctx, input, messageHandler)
	default:
		return r.streamPlanExecute(ctx, input, messageHandler)
	}
}

// Release 释放资源
func (r *RAGAgent) Release(ctx context.Context) {
	if retryModel, ok := r.chatModel.(*retryChatModel); ok {
		retryModel.Release(ctx)
	}
}

// runReAct 使用 ReAct 模式执行
func (r *RAGAgent) runReAct(ctx context.Context, input *schema.Message) (*schema.Message, error) {
	agent, err := r.BuildRAGAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.Generate(ctx, []*schema.Message{input})
}

// streamReAct 使用 ReAct 模式流式执行
func (r *RAGAgent) streamReAct(ctx context.Context, input *schema.Message) (*schema.StreamReader[*schema.Message], error) {
	agent, err := r.BuildRAGAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.Stream(ctx, []*schema.Message{input})
}

// runPlanExecute 使用 Plan-Execute 模式执行
func (r *RAGAgent) runPlanExecute(ctx context.Context, input *schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.Message, error) {
	agent, err := r.BuildPlanExecuteAgent(messageHandler)
	if err != nil {
		return nil, err
	}
	return agent.Run(ctx, input)
}

// streamPlanExecute 使用 Plan-Execute 模式流式执行
func (r *RAGAgent) streamPlanExecute(ctx context.Context, input *schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.StreamReader[*schema.Message], error) {
	agent, err := r.BuildPlanExecuteAgent(messageHandler)
	if err != nil {
		return nil, err
	}
	return agent.Stream(ctx, input)
}

func toolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.WithCtx(ctx).Error("stream reader error", err)
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
	return false, nil
}
