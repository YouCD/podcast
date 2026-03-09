package weekday_month

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/internal/ai/common"
	"podcast/pkg/types"
	"time"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
)

func rewriteUserQuery(ctx context.Context, userQuery string) (*graphState, error) {
	input := map[string]any{
		"date":      time.Now().Format("2006-01-02") + " " + time.Now().Weekday().String(),
		"userQuery": userQuery,
	}

	msgs, err := newGenQueryReWritingTemplate(ctx).Format(ctx, input)
	if err != nil {
		return nil, err
	}
	c, _, err := common.RunModelGenerate(ctx, "rewrite_user_query", msgs, openai.ChatCompletionResponseFormatTypeText, 5)
	if err != nil {
		return nil, err
	}

	// 将 Content 转换为 query
	var q types.UserQuery
	err = json.Unmarshal([]byte(c), &q)
	if err != nil {
		return nil, fmt.Errorf("QueryRewriting json unmarshal error: %w", err)
	}

	state := &graphState{
		userQueryStr: userQuery,
		userQueryRaw: c,
		userQuery:    &q,
	}
	return state, nil
}
