package weekday_month

import (
	"context"
	"podcast/internal/ai/common"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
)

func completion(ctx context.Context, model model.ToolCallingChatModel, input map[string]any, createPrompt func(ctx context.Context) prompt.ChatTemplate) (string, error) {
	msg, err := createPrompt(ctx).Format(ctx, input)
	if err != nil {
		return "", err
	}

	resp, err := common.ModelGenerate(ctx, model, msg)
	if err != nil {
		return "", err
	}

	output := strings.TrimSpace(resp)
	return output, nil
}
