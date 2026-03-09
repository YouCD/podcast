package main

import (
	"context"
	"os"
	"podcast/config"
	"podcast/internal/ai/llm"
	"podcast/internal/database/models"
	"podcast/pkg/rag"
	"sync"

	"github.com/youcd/toolkit/log"
)

func main() {
	id := os.Args[1]
	ctx := context.Background()
	c, err := config.LoadAppConfig("/root/podcast/config.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(ctx).Infof("%#v", c)
	models.Init()
	llmPool := llm.NewLLMPool()
	llmInfo, err := llmPool.Get(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("Get error: %w", err)
		return
	}
	engine, err := rag.NewEngine(ctx, llmInfo)
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine error: %w", err)
		return
	}
	var posts []*models.RssContent

	err = models.GetDb().Model(&models.RssContent{}).Where("id > ?", id).Find(&posts).Error
	if err != nil {
		log.WithCtx(ctx).Errorf("FindAll error: %s", err)
		os.Exit(1)
		return
	}

	pChan := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for _, content := range posts {
		wg.Add(1)
		go func(content *models.RssContent) {
			pChan <- struct{}{}
			defer func() {
				<-pChan
				wg.Done()
			}()
			if err := engine.AddRssContent(ctx, content); err != nil {
				log.WithCtx(ctx).Errorw("AddRssContent", "id", content.ID, "err", err)
				return
			}
			log.WithCtx(ctx).Infow("AddRssContent", "成功", content.ID)
		}(content)

	}
	wg.Wait()
}
