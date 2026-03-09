package embedding

import (
	"context"
	"fmt"

	"podcast/config"

	"github.com/sashabaranov/go-openai"
)

func New(ctx context.Context, content string) ([]openai.Embedding, error) {
	conf := openai.DefaultAnthropicConfig(config.Cfg.Vector.Embedding.APIKey, config.Cfg.Vector.Embedding.BaseURL)
	conf.APIType = openai.APITypeOpenAI
	eLLMCient := openai.NewClientWithConfig(conf)

	resp, err := eLLMCient.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Input:      content, // Channels are not serializable
			Model:      openai.EmbeddingModel(config.Cfg.Vector.Embedding.Model),
			Dimensions: 1024,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("解码响应时发生错误: %w", err)
	}

	return resp.Data, nil
}
