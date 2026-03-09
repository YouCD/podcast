package daily

import (
	"context"
	"encoding/json"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

func generateHTML(ctx context.Context, state *graphState) (*graphState, error) {
	if state.report.LLMResult != "" {
		log.WithCtx(ctx).Info("LLMResult 已存在")
		return state, nil
	}
	log.WithCtx(ctx).Info("开始生成HTML")
	format, err := newGenHtmlSummaryTemplate().Format(ctx, map[string]any{"content": state.report.Content})
	if err != nil {
		return state, err
	}
	var count int
Retry:
	llmResult, llmInfo, err := common.RunModelGenerate(ctx, "generateHTML", format, openai.ChatCompletionResponseFormatTypeJSONObject, 5)
	if err != nil {
		return state, err
	}
	var q types.LLMReport
	err = json.Unmarshal([]byte(llmResult), &q)
	if err != nil {
		log.WithCtx(ctx).Errorw("generateHTML", "provider", llmInfo, "llmResult", llmResult, "err", err)
		if count < 3 {
			count++
			goto Retry
		}
		// 如果解析失败，则将LLMResult置空
		state.report.LLMResult = ""
	} else {
		state.report.LLMResult = q.LLMResult
	}
	log.WithCtx(ctx).Info("生成HTML完成")
	return state, nil
}
