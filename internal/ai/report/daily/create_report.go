package daily

import (
	"context"
	"fmt"
	"strings"
	"time"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/youcd/toolkit/log"
)

func createReport(ctx context.Context, reportID int) (*graphState, error) {
	// 检查是否已存在报告
	existingReport, startDate, endDate, question, err := getExistingReport(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing report: %w", err)
	}
	if existingReport == nil { // 没有创建新的 报告
		return &graphState{ // 创建新报告
			report: &models.Report{
				TimeArray: fmt.Sprintf("%s~%s", startDate.Format("15:04:05"), endDate.Format("15:04:05")),
				Question:  question,
			},
			startDate:   startDate,
			endDate:     endDate,
			isNewReport: true,
		}, nil
	}
	return &graphState{
		report:    existingReport,
		startDate: startDate,
		endDate:   endDate,
	}, nil
}

func getExistingReport(ctx context.Context, id int) (*models.Report, time.Time, time.Time, string, error) {
	reportDao := dao.NewReportDao(models.GetDb())

	if id == 0 {
		var startDate, endDate time.Time
		var question string
		// 获取昨天的日期范围
		yesterday := time.Now().AddDate(0, 0, -1)
		startDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, yesterday.Location())
		question = fmt.Sprintf("日报-%s", yesterday.Format("2006-01-02"))

		byQuestion, err := reportDao.GetRepoByQuestion(ctx, question)
		if err == nil {
			log.WithCtx(ctx).Infof("已存在日报记录: %#v", byQuestion)
			return byQuestion, startDate, endDate, question, nil
		}
		return nil, startDate, endDate, question, nil
	}

	report, err := reportDao.GetReportByID(ctx, id)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, time.Time{}, time.Time{}, "", ErrNotExistingReport
	}
	var startTime, endTime time.Time
	split := strings.Split(report.Question, "日报-")
	if len(split) > 0 {
		startDate, err := time.Parse("2006-01-02", split[1])
		if err != nil {
			log.WithCtx(ctx).Error(err)
			return nil, time.Time{}, time.Time{}, report.Question, nil
		}
		startTime = startDate
		endTime = startTime.AddDate(0, 0, 1).Add(-time.Second)
	}
	return report, startTime, endTime, report.Question, nil
}
