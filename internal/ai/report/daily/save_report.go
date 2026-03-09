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
		err := reportDao.Create(ctx, state.report)
		if err != nil {
			return nil, fmt.Errorf("failed to save report: %w", err)
		}
		return state, nil
	}
	log.WithCtx(ctx).Info("更新报告")
	// 更新现有报告
	//nolint:modernize
	updates := make(map[string]interface{})

	if state.report.LLMResult != "" {
		updates["llm_result"] = state.report.LLMResult
	}
	if state.report.PodcastContent != "" {
		updates["podcast_content"] = state.report.PodcastContent
	}
	if state.report.PodcastMP3URL != "" {
		updates["podcast_mp3_url"] = state.report.PodcastMP3URL
	}
	if state.report.Content != "" {
		updates["content"] = state.report.Content
	}
	updates["time_array"] = state.report.TimeArray

	err := reportDao.UpdateByFields(ctx, dao.FieldKv{Field: "id", Value: state.report.ID}, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update report: %w", err)
	}
	log.WithCtx(ctx).Info("报告保存成功")
	return state, nil
}
