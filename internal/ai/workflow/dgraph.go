package workflow

import (
	"context"
	"encoding/json"
	"slices"
	"sync"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
	"golang.org/x/sync/semaphore"
)

func dgraph(ctx context.Context, state *graphState) ([]*types.RSSItem, error) {
	if len(state.LlmResult) == 0 {
		return nil, ErrNotLLMResult
	}
	const maxConcurrency = 2
	sem := semaphore.NewWeighted(maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var temp []*types.RSSItem
	for _, item := range state.LlmResult {
		if !analyzeDgraph(item.Categories) {
			mu.Lock()
			temp = append(temp, item)
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(i *types.RSSItem) {
			defer wg.Done()

			// 信号量控制并发
			if err := sem.Acquire(ctx, 1); err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "error", err)
				return
			}
			defer sem.Release(1)

			msgs, err := newRssDgraphTemplate(i.Categories).Format(ctx, map[string]any{
				"date":    i.Date,
				"content": i.Content,
				"title":   i.Title,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "error", err)
				return
			}
			llmResult, llmInfo, err := common.RunModelGenerate(ctx, state.llmPool, "dgraph", msgs, openai.ChatCompletionResponseFormatTypeJSONObject, 5)
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "provider", llmInfo, "md5", i.MD5, "content", i.Content, "error", err)
				return
			}
			var payload types.DgraphPayload
			err = json.Unmarshal([]byte(llmResult), &payload)
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "error", err)
				return
			}
			i.Dgraph = llmResult
			mu.Lock()
			temp = append(temp, i)
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	return temp, nil
}

var categories = []string{
	"财经",
	"行业洞察",
	"政治",
	"国际新闻",
	"科技",
	"AI",
}

func analyzeDgraph(c string) bool {
	return slices.Contains(categories, c)
}
