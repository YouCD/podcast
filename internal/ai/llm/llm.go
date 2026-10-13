package llm

import (
	"errors"

	"podcast/pkg/types"
)

var (
	// ErrSummaryNotConfigured 总结模型未配置
	ErrSummaryNotConfigured = errors.New("总结模型（llm.summary）未配置")
	// ErrNormalNotConfigured 普通模型未配置
	ErrNormalNotConfigured = errors.New("普通模型（llm.normal）未配置")
)

// Models 双模型管理：总结模型 + 普通模型。
//
// - Summary：总结模型，供报表（internal/ai/report）与智能体（internal/ai/agent）使用
// - Normal：普通模型，供其余请求（RSS 工作流、MCP、聊天标题等）使用
type Models struct {
	Summary *types.LLMInfo
	Normal  *types.LLMInfo
}

// NewModels 创建双模型实例，两者均不可为空
func NewModels(summary, normal *types.LLMInfo) (*Models, error) {
	if summary == nil {
		return nil, ErrSummaryNotConfigured
	}
	if normal == nil {
		return nil, ErrNormalNotConfigured
	}
	return &Models{Summary: summary, Normal: normal}, nil
}
