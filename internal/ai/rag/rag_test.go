package rag

import (
	"context"
	"fmt"
	"io"
	"testing"

	"podcast/config"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func init() {
	_, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	if err != nil {
		panic(err)
	}
}

func TestEngine_Query(t *testing.T) {
	ctx := context.Background()
	llmInfo := config.Cfg.LLM.Normal
	// 创建 RAG 引擎配置
	ragCfg := &types.RagConfig{
		PgVector: config.Cfg.Database.PostgreSQL.ToPgVector(),
		Embedding: config.Cfg.Vector.Embedding,
	}
	ragCli, err := NewEngine(ctx, llmInfo, ragCfg)
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
