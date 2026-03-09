package main

import (
	"context"
	"fmt"
	"io"
	"podcast/config"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"

	"github.com/youcd/toolkit/log"
)

func main() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.local.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)

	ctx := context.Background()
	llmPool := llm.NewLLMPool()
	get, err := llmPool.Get(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("Get error: %w", err)
		return
	}
	ragCli, err := rag.NewEngine(ctx, get)
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
