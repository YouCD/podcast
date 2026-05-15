package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
	"golang.org/x/sync/semaphore"
)

//nolint:gocognit
func analyzeRss(ctx context.Context, state *graphState) (*graphState, error) {
	if len(state.Categorization) == 0 {
		return state, nil
	}

	const maxConcurrency = 4
	sem := semaphore.NewWeighted(maxConcurrency)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var rssItems []*types.RSSItem
	for item, s := range state.Categorization {
		item.Categories = s
		rssItems = append(rssItems, item)
	}

	for _, item := range rssItems {
		wg.Add(1)
		go func(i *types.RSSItem) {
			defer wg.Done()

			// 信号量控制并发
			if err := sem.Acquire(ctx, 1); err != nil {
				mu.Lock()
				state.Errors = append(state.Errors, fmt.Errorf("sem acquire failed for %s: %w", i.Title, err))
				mu.Unlock()
				return
			}
			defer sem.Release(1)
			msgs, err := newRssAnalyzeTemplate(ctx, item.Categories).Format(ctx, map[string]any{
				"content": i.Content,
				"date":    time.Now(),
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("analyze_rss", "error", err)
				return
			}
			llmResult, llmInfo, err := common.RunModelGenerate(ctx, state.llmPool, "analyze_rss", msgs, openai.ChatCompletionResponseFormatTypeJSONObject, 5)
			if err != nil {
				log.WithCtx(ctx).Errorw("analyze_rss", "provider", llmInfo, "error", err)
				return
			}

			var result types.LLMResult
			err = json.Unmarshal([]byte(llmResult), &result)
			if err != nil {
				log.WithCtx(ctx).Errorw("analyze_rss", "provider", llmInfo, "title", i.Title, "md5", i.MD5, "error", err)
				return
			}
			marshal, _ := json.Marshal(result)
			i.LLMResult = string(marshal)
			log.WithCtx(ctx).Debugw("analyzeRss", "provider", llmInfo, "md5", i.MD5, "content", string(marshal))
		}(item)
	}
	wg.Wait()

	for _, item := range rssItems {
		if item.LLMResult != "" {
			state.LlmResult = append(state.LlmResult, item)
		}
	}
	return state, nil // 始终返回 nil，单个失败不阻塞整体流程
}
