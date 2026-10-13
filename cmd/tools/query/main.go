package main

import (
	"context"
	"fmt"
	"io"

	"podcast/config"
	"podcast/internal/ai/rag"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func main() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.local.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)

	ctx := context.Background()
	// 使用普通模型
	llmInfo := c.LLM.Normal
	ragCfg := &types.RagConfig{
		PgVector: &types.PgVector{
			Host:          c.Database.PostgreSQL.Host,
			Port:          c.Database.PostgreSQL.Port,
			User:          c.Database.PostgreSQL.User,
			Password:      c.Database.PostgreSQL.Password,
			DBName:        c.Database.PostgreSQL.DBName,
			RssCollection: c.Database.PostgreSQL.RssCollection,
		},
		Embedding: &types.Embedding{
			APIKey: c.Vector.Embedding.APIKey,
			Model:  c.Vector.Embedding.Model,
		},
	}
	ragCli, err := rag.NewEngine(ctx, llmInfo, ragCfg)
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine error: %w", err)
		return
	}

	// 检索
	var query string
	for {
		_, _ = fmt.Scan(&query)
		rsp, err := ragCli.Query(ctx, query)
		if err != nil {
			log.WithCtx(ctx).Errorf("Query error: %w", err)
			return
		}

		// 流式输出
		for {
			output, err := rsp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.WithCtx(ctx).Errorf("Recv error: %w", err)
			}
			fmt.Print(output.Content)
		}
	}
}
