package agent

import (
	"context"
	"errors"
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

// RAGAgentConfig RAG Agent 配置
type RAGAgentConfig struct {
	MCP       *MCPConfig
	RagConfig *types.RagConfig
}
type RAGAgent struct {
	chatModel model.ToolCallingChatModel
	tools     []tool.BaseTool
}

func NewAgent(ctx context.Context, cfg *RAGAgentConfig) (*RAGAgent, error) {
	// 初始化 MCP 工具
	tm, err := NewToolManager(ctx, cfg.MCP, cfg.RagConfig)
	if err != nil {
		return nil, err
	}

	// 创建 RAG 引擎
	newRetryChatModel, err := NewRetryChatModel(ctx, 5, defaultBackoff, openai.ChatCompletionResponseFormatTypeJSONObject)
	if err != nil {
		log.WithCtx(ctx).Error("Failed to init retry model", err)
		return nil, err
	}
	defer newRetryChatModel.Release(ctx)

	// 创建 RAG 引擎
	m, err := NewRetryChatModel(ctx, 5, defaultBackoff, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		log.WithCtx(ctx).Error("Failed to init retry model", err)
		return nil, err
	}
	return &RAGAgent{
		chatModel: m,
		tools:     tm.GetTools(),
	}, nil
}

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
