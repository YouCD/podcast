package workflow

import (
	"context"
	"time"

	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

// filterToday 筛选今日内容
func filterToday(ctx context.Context, state *graphState) (*graphState, error) {
	log.WithCtx(ctx).Info("[阶段2/7] 开始筛选今日内容")
	today := time.Now().Truncate(24 * time.Hour)
	var output []*types.RSSItem
	for _, item := range state.RawItems {
		log.WithCtx(ctx).Debugw("filterToday", "Source", item.Source, "MD5", item.MD5, "Title", item.Title, "Date", item.Date.Format("2006-01-02 15:04:05"))
		// 只保留今天的数据
		if item.Date.Truncate(24 * time.Hour).Equal(today) {
			output = append(output, item)
		}
	}
	state.Filtered = output
	log.WithCtx(ctx).Infof("[阶段2/7] 筛选完成，共 %d 条", len(output))
	return state, nil
}
