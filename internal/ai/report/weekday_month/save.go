package weekday_month

import (
	"context"
	"fmt"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/youcd/toolkit/log"
)

func save(ctx context.Context, state *graphState) (*graphState, error) {
	reportDao := dao.NewReportDao(models.GetDb())
	report := models.Report{
		Question:  state.userQueryStr,
		TimeArray: state.userQuery.StartTime + "/" + state.userQuery.EndTime,
		LLMResult: state.llmResult,
		Genre:     2,
	}
	err := reportDao.Create(ctx, &report)
	if err != nil {
		log.WithCtx(ctx).Errorw("LLMReport", "err", err)
		return state, fmt.Errorf("create err: %w", err)
	}
	return state, nil
}
