package daily

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"podcast/internal/ai/common"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

func generateContent(ctx context.Context, state *graphState) (*graphState, error) {
	if state.report.Content != "" {
		log.WithCtx(ctx).Info("Markdown 内容已存在")
		return state, nil
	}
	if state.isNewReport {
		if len(state.rssContents) == 0 {
			return state, ErrNotExistingRssContent
		}
	}
	log.WithCtx(ctx).Info("Markdown 内容生成开始")
	// 拼接 RSS 内容
	var concatenatedContent strings.Builder
	for _, rss := range state.rssContents {
		concatenatedContent.WriteString(fmt.Sprintf("[标题]：%s\n", rss.Title))
		content := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(rss.Content), " ")
		concatenatedContent.WriteString(fmt.Sprintf("[内容]：%s\n", content))
		concatenatedContent.WriteString(fmt.Sprintf("[链接]：%s\n\n", rss.Link))
	}
	log.WithCtx(ctx).Debugw("generateContent", "total Contents", len(state.rssContents))

	format, err := newGenMarkdownSummaryTemplate().Format(ctx, map[string]any{"content": concatenatedContent.String()})
	if err != nil {
		return state, err
	}
	var count int
Retry:
	llmResult, llmInfo, err := common.RunModelGenerate(ctx, "generateContent", format, openai.ChatCompletionResponseFormatTypeText, 5)
	if err != nil {
		return state, err
	}

	if len([]rune(llmResult)) < 100 {
		count++
		if count < 3 {
			log.WithCtx(ctx).Warnw("generateContent", "provider", llmInfo, "llmResult", llmResult)
			goto Retry
		}
	}
	if len([]rune(llmResult)) < 100 {
		return state, ErrContentIsToShort
	}
	state.report.Content = llmResult
	log.WithCtx(ctx).Infof("Markdown 内容生成完成")
	return state, nil
}
