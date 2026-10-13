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
	if state.Report.Content != "" {
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
	llmResult, llmInfo, err := common.RunModelGenerateReAct(ctx, state.llmInfo, "generateContent", format,
		func(output string) (string, error) {
			if len([]rune(output)) < 100 {
				return "", fmt.Errorf("%w: 仅 %d 字，请基于给定素材写出完整的日报（不少于100字）", ErrContentIsToShort, len([]rune(output)))
			}
			return StripThinkContent(output), nil
		},
		3, 5, 0, false, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		log.WithCtx(ctx).Errorw("generateContent", "provider", llmInfo, "llmResult", llmResult, "err", err)
		return state, err
	}
	state.Report.Content = llmResult
	log.WithCtx(ctx).Infof("Markdown 内容生成完成")
	return state, nil
}

// thinkRegex 匹配 <think>...</think> 及其内部内容
// (?s) 让 . 可以匹配换行符
var thinkRegex = regexp.MustCompile(`(?s)<think>.*?</think>`)

// StripThinkContent 去除文本中的思考内容（<think>...</think>）
// 同时会清理首尾多余的空白字符
func StripThinkContent(content string) string {
	// 1. 移除所有 <think>...</think> 块
	cleaned := thinkRegex.ReplaceAllString(content, "")

	// 2. 如果存在不完整的 <think>（没有闭合标签），也一并处理
	if idx := strings.Index(cleaned, "<think>"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	// 3. 清理首尾空白
	return strings.TrimSpace(cleaned)
}
