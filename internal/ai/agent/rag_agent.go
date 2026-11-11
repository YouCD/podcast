package agent

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	LLMInfo   *types.LLMInfo // 总结模型（agent 专用）
	// Agent 模式，默认为 Plan-Execute
	Mode AgentMode
	// 最大重规划次数（仅 Plan-Execute 模式有效）
	MaxReplan int
	// 内部工具策略配置文件路径（agent_tools.yaml），可为空
	ToolsFile string
	// 外部 MCP 代理配置（mcpProxy），工具策略来源之一
	MCPProxy map[string]*types.Mcp
}

// RAGAgent RAG Agent
type RAGAgent struct {
	chatModel   model.ToolCallingChatModel
	tools       []tool.BaseTool
	tm          *ToolManager // 工具管理器（用于释放 MCP/数据库资源）
	mode        AgentMode
	maxReplan   int
	strategySrc *types.ToolStrategySource // 工具策略来源（内部配置 + mcpProxy）
}

// NewAgent 创建 Agent（默认使用 Plan-Execute 模式）
func NewAgent(ctx context.Context, cfg *RAGAgentConfig) (*RAGAgent, error) {
	// 初始化 MCP 工具
	tm, err := NewToolManager(ctx, cfg.MCP, cfg.RagConfig)
	if err != nil {
		return nil, err
	}

	// 创建 ChatModel（agent 使用总结模型）
	m, err := NewRetryChatModel(ctx, cfg.LLMInfo, 5, defaultBackoff, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		log.WithCtx(ctx).Error("Failed to init retry model", err)
		return nil, err
	}

	// 加载工具策略配置（内部工具 agent_tools.yaml + 外部 mcpProxy）
	internalTools, err := types.LoadAgentToolsConfig(cfg.ToolsFile)
	if err != nil {
		return nil, fmt.Errorf("加载工具策略配置失败: %w", err)
	}
	strategySrc := &types.ToolStrategySource{
		Internal: internalTools,
		MCPProxy: cfg.MCPProxy,
	}

	// 设置默认模式
	mode := cfg.Mode
	if mode == "" {
		mode = ModePlanExecute
	}

	return &RAGAgent{
		chatModel:   m,
		tools:       tm.GetTools(),
		tm:          tm,
		mode:        mode,
		maxReplan:   cfg.MaxReplan,
		strategySrc: strategySrc,
	}, nil
}

// Close 释放 Agent 持有的资源（MCP 客户端、PgVector 连接等）
func (r *RAGAgent) Close(ctx context.Context) error {
	if r.tm != nil {
		return r.tm.Close(ctx)
	}
	return nil
}

// BuildRAGAgent 构建 ReAct Agent（保留原有接口兼容）
func (r *RAGAgent) BuildRAGAgent(ctx context.Context) (*react.Agent, error) {
	// 动态生成工具详解与使用策略段落
	toolsSection := r.buildToolsSection(ctx)

	// 使用Eino内置的ReAct Agent
	config := &react.AgentConfig{
		ToolCallingModel: r.chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: r.tools},
		// 最大迭代次数，防止无限循环
		MaxStep: 20,
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(fmt.Sprintf(reactPromptTemplate, toolsSection))}, input...)
		},
		StreamToolCallChecker: toolCallChecker,
	}
	return react.NewAgent(ctx, config)
}

// buildToolsSection 基于已注册工具和策略来源，动态生成 ReAct 提示词的工具详解段落
func (r *RAGAgent) buildToolsSection(ctx context.Context) string {
	var infos []*schema.ToolInfo
	for _, t := range r.tools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	p := &Planner{strategySrc: r.strategySrc}
	return p.buildToolsDescription(infos)
}

// BuildPlanExecuteAgent 构建 Plan-Execute Agent
func (r *RAGAgent) BuildPlanExecuteAgent(messageHandler func(ctx context.Context, stage string, data map[string]any)) (*PlanExecuteAgent, error) {
	config := &PlanExecuteAgentConfig{
		ChatModel:     r.chatModel,
		Tools:         r.tools,
		MaxReplan:     r.maxReplan,
		MessageHandle: messageHandler,
		StrategySrc:   r.strategySrc,
	}
	return NewPlanExecuteAgent(config)
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
// history 为多轮对话历史（不含当前问题，按时间正序）
func (r *RAGAgent) Run(ctx context.Context, input *schema.Message, history []*schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.Message, error) {
	switch r.mode {
	case ModeReAct:
		return r.runReAct(ctx, input, history)
	case ModePlanExecute:
		return r.runPlanExecute(ctx, input, history, messageHandler)
	default:
		return r.runPlanExecute(ctx, input, history, messageHandler)
	}
}

// Stream 流式执行 Agent
// history 为多轮对话历史（不含当前问题，按时间正序）
func (r *RAGAgent) Stream(ctx context.Context, input *schema.Message, history []*schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.StreamReader[*schema.Message], error) {
	switch r.mode {
	case ModeReAct:
		return r.streamReAct(ctx, input, history)
	case ModePlanExecute:
		return r.streamPlanExecute(ctx, input, history, messageHandler)
	default:
		return r.streamPlanExecute(ctx, input, history, messageHandler)
	}
}

// runReAct 使用 ReAct 模式执行
func (r *RAGAgent) runReAct(ctx context.Context, input *schema.Message, history []*schema.Message) (*schema.Message, error) {
	agent, err := r.BuildRAGAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.Generate(ctx, append(append([]*schema.Message{}, history...), input))
}

// streamReAct 使用 ReAct 模式流式执行
func (r *RAGAgent) streamReAct(ctx context.Context, input *schema.Message, history []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	agent, err := r.BuildRAGAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.Stream(ctx, append(append([]*schema.Message{}, history...), input))
}

// runPlanExecute 使用 Plan-Execute 模式执行
func (r *RAGAgent) runPlanExecute(ctx context.Context, input *schema.Message, history []*schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.Message, error) {
	agent, err := r.BuildPlanExecuteAgent(messageHandler)
	if err != nil {
		return nil, err
	}
	return agent.Run(ctx, input, history)
}

// streamPlanExecute 使用 Plan-Execute 模式流式执行
func (r *RAGAgent) streamPlanExecute(ctx context.Context, input *schema.Message, history []*schema.Message, messageHandler func(ctx context.Context, stage string, data map[string]any)) (*schema.StreamReader[*schema.Message], error) {
	agent, err := r.BuildPlanExecuteAgent(messageHandler)
	if err != nil {
		return nil, err
	}
	return agent.Stream(ctx, input, history)
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
