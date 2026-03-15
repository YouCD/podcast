package daily

import (
	"context"

	"podcast/internal/ai/common"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

func generatePodcastContent(ctx context.Context, state *graphState) (*graphState, error) {
	if state.report.PodcastContent != "" {
		log.WithCtx(ctx).Info("播客内容已存在")
		return state, nil
	}
	if state.isNewReport {
		if state.report.Content == "" {
			return state, ErrContentIsEmpty
		}
	}
	log.WithCtx(ctx).Info("播客内容生成开始")
	format, err := newGenPodcastSummaryTemplate().Format(ctx, map[string]any{"content": state.report.Content})
	if err != nil {
		return state, err
	}

	llmResult, llmInfo, err := common.RunModelGenerate(ctx, "generatePodcastContent", format, openai.ChatCompletionResponseFormatTypeText, 5)
	if err != nil {
		return state, err
	}
	state.report.PodcastContent = llmResult
	log.WithCtx(ctx).Infow("播客内容生成结束", "llmInfo", llmInfo)
	return state, nil
}
