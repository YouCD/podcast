package weekday_month

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"strings"
	"time"

	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

type content struct {
	Content  string      `json:"content"`
	Subtitle string      `json:"key_points"`
	QA       []*types.QA `json:"qa"`
}

func (c *content) FmtStr() string {
	var qaStr strings.Builder
	for _, i2 := range c.QA {
		fmt.Fprintf(&qaStr, "  %s : %s\n", i2.Question, i2.Answer)
	}

	return fmt.Sprintf(`content: %s
subtitle: %s
QA: 
%s
`, c.Content, c.Subtitle, qaStr.String())
}

func getRssContent(ctx context.Context, state *graphState) (*graphState, error) {
	if state.userQuery == nil {
		return nil, fmt.Errorf("userQuery is nil")
	}
	startTime, err := time.Parse("2006-01-02", state.userQuery.StartTime)
	if err != nil {
		log.WithCtx(ctx).Errorw("time.Parse", "time", state.userQuery.StartTime, "err", err)
		return nil, fmt.Errorf("time.Parse err: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", state.userQuery.EndTime)
	if err != nil {
		log.WithCtx(ctx).Errorw("time.Parse", "time", state.userQuery.EndTime, "err", err)
		return nil, fmt.Errorf("time.Parse err: %w", err)
	}
	rssContentDao := dao.NewRssContentDao(models.GetDb())

	dateRange, err := rssContentDao.FindByDateRangeCategories(ctx, startTime, endTime, state.userQuery.Categories)
	if err != nil {
		log.WithCtx(ctx).Errorw("FindByDateRange", "startTime", startTime, "endTime", endTime, "err", err)
		return nil, fmt.Errorf("FindByDateRange err: %w", err)
	}

	for _, c := range dateRange {
		var llmResult types.LLMResult
		err = json.Unmarshal([]byte(c.LLMResult), &llmResult)
		if err != nil {
			log.WithCtx(ctx).Errorw("json.Unmarshal", "content", c.Content, "err", err)
			continue
		}

		state.contents = append(state.contents, &content{
			Content:  c.Content,
			Subtitle: llmResult.Subtitle,
			QA:       llmResult.QA,
		})
	}

	return state, nil
}
