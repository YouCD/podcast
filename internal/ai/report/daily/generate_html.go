package daily

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

// stripCodeFences 剥离 LLM 返回内容外的 markdown 代码围栏，
// 如 ```html ... ``` 或 ```json ... ```、``` ... ```。
func stripCodeFences(s string) string {
	trimmed := strings.TrimSpace(s)
	const fence = "```"
	openIdx := strings.Index(trimmed, fence)
	if openIdx < 0 {
		return trimmed
	}
	// 内容起始于首行围栏之后（跳过语言标记，如 ```html）
	start := openIdx + len(fence)
	if eol := strings.IndexByte(trimmed[start:], '\n'); eol >= 0 {
		start += eol + 1
	}
	endIdx := strings.LastIndex(trimmed[start:], fence)
	if endIdx >= 0 {
		return strings.TrimSpace(trimmed[start : start+endIdx])
	}
	return strings.TrimSpace(trimmed[start:])
}

func generateHTML(ctx context.Context, state *graphState) (*graphState, error) {
	if state.Report.LLMResult != "" {
		log.WithCtx(ctx).Info("LLMResult 已存在")
		return state, nil
	}
	log.WithCtx(ctx).Info("开始生成HTML")
	format, err := newGenHtmlSummaryTemplate().Format(ctx, map[string]any{"content": state.Report.Content})
	if err != nil {
		return state, err
	}
	llmResult, llmInfo, err := common.RunModelGenerateReAct(ctx, state.llmInfo, "generateHTML", format,
		func(output string) (string, error) {
			output = stripCodeFences(output)
			var q types.LLMReport
			if err := json.Unmarshal([]byte(output), &q); err != nil {
				return "", fmt.Errorf("输出不是合法 JSON: %v，请直接返回合法 JSON（不要用 markdown 代码块包裹）", err)
			}
			if !strings.HasPrefix(q.LLMResult, "<!DOCTYPE html>") {
				return "", fmt.Errorf("llm_result 未以 <!DOCTYPE html> 开头，请返回完整的 HTML 文档")
			}
			return q.LLMResult, nil
		},
		3, 5, 0, false, openai.ChatCompletionResponseFormatTypeJSONObject)
	if err != nil {
		log.WithCtx(ctx).Errorw("generateHTML", "provider", llmInfo, "llmResult", llmResult, "err", err)
		// 如果解析失败，则将LLMResult置空
		state.Report.LLMResult = ""
		return state, nil
	}
	state.Report.LLMResult = llmResult
	log.WithCtx(ctx).Info("生成HTML完成")
	return state, nil
}
