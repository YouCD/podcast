package daily

import (
	"context"
	"fmt"
	"time"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/youcd/toolkit/log"
)

func rssContents(ctx context.Context, state *graphState) (*graphState, error) {
	if state.report.Content != "" {
		log.WithCtx(ctx).Info("Markdown 内容已存在")
		return state, nil
	}
	rssContentDao := dao.NewRssContentDao(models.GetDb())
	log.WithCtx(ctx).Infow("rssContents", "时间范围", fmt.Sprintf("开始获取RSS内容 %s~%s", state.startDate.Format(time.DateTime), state.endDate.Format(time.DateTime)))
	// 获取昨天的所有RSS内容
	rssContentList, err := rssContentDao.FindByDateRange(ctx, state.startDate, state.endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS contents: %w", err)
	}

	if len(rssContentList) == 0 {
		log.WithCtx(ctx).Infow("rssContents", "时间范围", fmt.Sprintf("%s~%s", state.startDate.Format(time.DateTime), state.endDate.Format(time.DateTime)), "RSS总数", 0)
		return nil, ErrNotExistingRssContent
	}
	state.rssContents = rssContentList
	return state, nil
}
