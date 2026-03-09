package weekday_month

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/internal/ai/common"
	"podcast/pkg/types"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

func analysis(ctx context.Context, state *graphState) (*graphState, error) {
	var contentAll strings.Builder
	if len(state.contents) == 0 {
		return state, fmt.Errorf("no content")
	}
	for _, i2 := range state.contents {
		fmt.Fprintf(&contentAll, "%s", i2.FmtStr())
	}

	msgs, err := newAnalysisTemplate(ctx, state.userQueryRaw).Format(ctx, map[string]any{
		"date":      time.Now(),
		"userQuery": state.userQueryStr,
		"content":   contentAll.String(),
	})
	if err != nil {
		return state, err
	}
	Content, llmInfo, err := common.RunModelGenerate(ctx, "analysis", msgs, openai.ChatCompletionResponseFormatTypeJSONObject, 5)
	if err != nil {
		return state, err
	}

	var q types.LLMReport
	err = json.Unmarshal([]byte(Content), &q)
	if err != nil {
		log.WithCtx(ctx).Errorw("analysis", "provider", llmInfo, "Content", Content, "err", err)
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}
	state.llmResult = q.LLMResult
	return state, nil
}
