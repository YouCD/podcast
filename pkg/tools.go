package pkg

import (
	"context"
	"fmt"
	"time"

	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
)

func WithRetry(fn func() error, maxRetries int) error {
	var err error
	for range maxRetries {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second) // 添加延迟避免频繁重试
	}
	return err
}

func NewChatModel(ctx context.Context, llmInfo *types.LLMInfo, responseFormat openai.ChatCompletionResponseFormatType) (*qwen.ChatModel, error) {
	// t := float32(0.3)
	enableThinking := false // 默认关闭深度思考，RSS 批量分类/分析对延迟敏感，可显著提速
	cfg := &qwen.ChatModelConfig{
		APIKey:  llmInfo.ApiKey,
		BaseURL: llmInfo.BaseURL,
		Model:   llmInfo.Model,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: responseFormat,
		},
		EnableThinking: &enableThinking,
		// Temperature: &t,
	}

	chatModel, err := qwen.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	return chatModel, nil
}
