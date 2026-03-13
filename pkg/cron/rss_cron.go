package cron

import (
	"context"
	"errors"
	"sync"
	"time"

	"podcast/pkg/types"

	"podcast/internal/ai/report/daily"
	"podcast/internal/ai/report/weekday_month"
	"podcast/internal/ai/workflow"

	"github.com/robfig/cron/v3"
	"github.com/youcd/toolkit/log"
)

var rssProcessingMutex sync.Mutex // 用于保证同一时间只有一个 RSS 处理任务运行

// StartRSSCronJob 启动定时爬取 RSS 任务
func StartRSSCronJob(ctx context.Context, cfg *types.RagConfig, RSS []*types.RSSSource) {
	// 获取应用配置中的 RSS 配置
	c := cron.New()

	// 创建处理器
	workFlow, err := workflow.New(ctx, cfg)
	if err != nil {
		log.WithCtx(ctx).Errorf("创建处理器失败: %v", err)
		return
	}
	// 每小时执行一次RSS爬取任务
	_, _ = c.AddFunc("*/10 * * * *", func() {
		if !rssProcessingMutex.TryLock() {
			log.WithCtx(ctx).Warn("RSS 爬取任务仍在执行中，跳过此次执行")
			return
		}
		defer rssProcessingMutex.Unlock()

		//nolint:all
		ctx = log.SetRequestId(ctx)
		log.WithCtx(ctx).Info("开始执行RSS爬取任务...")
		_, err := workFlow.Invoke(ctx, RSS)
		if err != nil {
			if errors.Is(err, workflow.ErrRSSSourceNotContent) || errors.Is(err, workflow.ErrNotLLMResult) {
				log.WithCtx(ctx).Warnf("此次RSS源抓取内容为空，err: %s", err.Error())
				return
			}
			log.WithCtx(ctx).Errorf("RSS爬取任务失败: %v", err)
			return
		}
		log.WithCtx(ctx).Info("RSS爬取任务执行完成")
	})
	c.Start()
	// 立即执行一次
	go func() {
		if !rssProcessingMutex.TryLock() {
			log.WithCtx(ctx).Warn("RSS 爬取任务仍在执行中，跳过此次执行")
			return
		}
		defer rssProcessingMutex.Unlock()

		//nolint:all
		ctx = log.SetRequestId(ctx)
		log.WithCtx(ctx).Info("首次执行RSS爬取任务...")
		output, err := workFlow.Invoke(ctx, RSS)
		if err != nil {
			if errors.Is(err, workflow.ErrRSSSourceNotContent) {
				log.WithCtx(ctx).Warn("此次RSS源抓取内容为空")
				return
			}
			log.WithCtx(ctx).Errorf("RSS爬取任务失败: %v", err)
		}
		log.WithCtx(ctx).Infof("首次RSS爬取任务执行完成, 数量: %d", len(output))
	}()

	log.WithCtx(ctx).Info("RSS定时爬取任务已启动")
}

// StartReportJob 启动定时报告任务
func StartReportJob(ctx context.Context, reports []*types.Report, podcast *types.Podcast) {
	c := cron.New()
	report, err := weekday_month.New(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("创建周月报告处理器失败: %v", err)
		return
	}
	for _, r := range reports {
		// 每小时执行一次RSS爬取任务
		_, _ = c.AddFunc(r.Schedule, func() {
			//nolint:all
			ctx = log.SetRequestId(ctx)
			now := time.Now()
			log.WithCtx(ctx).Infof("开始执行分析报告任务，topic:%s ", r.Topic)
			_, err := report.Invoke(ctx, r.Topic)
			if err != nil {
				log.WithCtx(ctx).Errorf("分析报告任务失败，已达最大重试次数: %v", err)
				return
			}
			log.WithCtx(ctx).Infof("结束执行分析报告任务，topic:%s ,持续时间：%s", r.Topic, time.Since(now))
		})
	}

	// 添加每日报告任务，每天凌晨0点执行

	dailyReport, err := daily.New(ctx, podcast)
	if err != nil {
		log.WithCtx(ctx).Errorf("创建每日报告处理器失败: %v", err)
		return
	}
	_, _ = c.AddFunc("0 0 * * *", func() {
		now := time.Now()
		//nolint:all
		ctx = log.SetRequestId(ctx)
		log.WithCtx(ctx).Info("开始执行每日报告任务...")
		_, err = dailyReport.Invoke(ctx, 0)
		if err != nil {
			log.WithCtx(ctx).Errorf("每日报告任务失败,持续时间：%s, err: %s", time.Since(now), err)
			return
		}
		log.WithCtx(ctx).Infof("每日报告任务执行完成，持续时间：%s", time.Since(now))
	})

	c.Start()
}
