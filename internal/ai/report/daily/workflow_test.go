package daily

import (
	"context"
	"testing"
	"time"

	"podcast/internal/ai/llm"

	"podcast/config"
	"podcast/internal/database/models"

	"github.com/youcd/toolkit/log"
)

func init() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.local.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)
	_, err = models.Init(c)
	if err != nil {
		panic(err)
	}
	llm.NewLLMPool(config.Cfg.LLM)
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	dailyReport, err := New(ctx, config.Cfg.Podcast)
	if err != nil {
		panic(err)
	}
	now := time.Now()
	_, err = dailyReport.Invoke(ctx, 0)
	if err != nil {
		log.WithCtx(ctx).Errorf("每日报告任务失败,持续时间：%s, err: %s", time.Since(now), err)
		return
	}
	log.WithCtx(ctx).Infof("每日报告任务执行完成，持续时间：%s", time.Since(now))
}
