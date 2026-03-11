package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/pkg/types"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
)

func ConversationTitle(ctx context.Context, llm *types.LLMInfo, msg []*types.MessageInfo) (string, error) {
	chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: llm.BaseURL,
		APIKey:  llm.ApiKey,
		Model:   llm.Model,
	})
	if err != nil {
		return "", fmt.Errorf("NewChatModel error: %w", err)
	}
	marshal, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("json.Marshal error: %w", err)
	}
	resp, err := chatModel.Generate(ctx, []*schema.Message{
		schema.UserMessage(fmt.Sprintf(PromptConversationTitle, string(marshal))),
	})
	if err != nil {
		return "", fmt.Errorf("Generate error: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}
