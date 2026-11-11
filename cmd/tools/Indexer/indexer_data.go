package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"podcast/config"
	"podcast/internal/ai/rag"
	"podcast/internal/database/models"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func main() {
	// 用法: indexer <起始id> [config路径]
	if len(os.Args) < 2 {
		fmt.Println("用法: indexer <起始id> [config路径]")
		os.Exit(1)
	}
	idStr := os.Args[1]
	configPath := "/root/podcast/config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}
	ctx := context.Background()
	c, err := config.LoadAppConfig(configPath)
	if err != nil {
		panic(err)
	}
	log.WithCtx(ctx).Infof("%#v", c)
	log.WithCtx(ctx).Infof("%#v", c)
	_, err = models.Init(c)
	if err != nil {
		panic(err)
	}

	// 使用普通模型

	// 创建 RAG 引擎配置
	ragCfg := &types.RagConfig{
		PgVector: &types.PgVector{
			Host:          c.Database.PostgreSQL.Host,
			Port:          c.Database.PostgreSQL.Port,
			User:          c.Database.PostgreSQL.User,
			Password:      c.Database.PostgreSQL.Password,
			DBName:        c.Database.PostgreSQL.DBName,
			RssCollection: c.Database.PostgreSQL.RssCollection,
			Dimension:     c.Vector.Embedding.Dimension,
		},
		Embedding: c.Vector.Embedding,
	}

	engine, err := rag.NewEngine(ctx, ragCfg)
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine error: %w", err)
		return
	}
	var posts []*models.RssContent

	if idStr == "missing" {
		// 只补未建向量的文章（md5 不在向量表中）
		err = models.GetDb().Model(&models.RssContent{}).
			Where("(categories IS NULL OR categories != 'low_quality') AND md5 NOT IN (SELECT DISTINCT md5 FROM rss_vectors)").
			Order("id").Find(&posts).Error
	} else {
		startID, aerr := strconv.Atoi(idStr)
		if aerr != nil {
			fmt.Println("起始id必须为数字或 missing:", aerr)
			os.Exit(1)
		}
		err = models.GetDb().Model(&models.RssContent{}).Where("id > ? AND (categories IS NULL OR categories != 'low_quality')", startID).Find(&posts).Error
	}
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
