package daily

import (
	"context"
	"fmt"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/youcd/toolkit/log"
)

func saveReport(ctx context.Context, state *graphState) (*graphState, error) {
	reportDao := dao.NewReportDao(models.GetDb())
	if state.isNewReport {
		log.WithCtx(ctx).Info("保存新报告")
		// 保存到report表
		err := reportDao.Create(ctx, state.Report)
		if err != nil {
			return nil, fmt.Errorf("failed to save report: %w", err)
		}
		return state, nil
	}
	log.WithCtx(ctx).Info("更新报告")
	// 更新现有报告
	//nolint:modernize
	updates := make(map[string]interface{})

	if state.Report.LLMResult != "" {
		updates["llm_result"] = state.Report.LLMResult
	}
	if state.Report.PodcastContent != "" {
		updates["podcast_content"] = state.Report.PodcastContent
	}
	if state.Report.PodcastMP3URL != "" {
		updates["podcast_mp3_url"] = state.Report.PodcastMP3URL
	}
	if state.Report.Content != "" {
		updates["content"] = state.Report.Content
	}
	updates["time_array"] = state.Report.TimeArray

	err := reportDao.UpdateByFields(ctx, dao.FieldKv{Field: "id", Value: state.Report.ID}, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update report: %w", err)
	}
	log.WithCtx(ctx).Info("报告保存成功")
	return state, nil
}
