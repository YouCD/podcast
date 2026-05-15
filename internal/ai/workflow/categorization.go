package workflow

import (
	"context"
	"fmt"
	"sync"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
	"golang.org/x/sync/semaphore"
)

func categorization(ctx context.Context, state *graphState) (*graphState, error) {
	if len(state.UniqueItems) == 0 {
		return state, ErrRSSSourceNotContent
	}

	const maxConcurrency = 4
	sem := semaphore.NewWeighted(maxConcurrency)

	// 改用 WaitGroup 替代 errgroup，避免单点失败导致整体崩溃
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range state.UniqueItems {
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
			msgs, err := newCategorizationTemplate(ctx).Format(ctx, map[string]any{
				"title":   i.Title,
				"content": i.Content,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("categorization", "md5", i.MD5, "content", i.Content, "error", err)
				state.Categorization[i] = "其他"
				return
			}
			var count int
		Retry:
			category, llmInfo, err := common.RunModelGenerate(ctx, state.llmPool, "categorization", msgs, openai.ChatCompletionResponseFormatTypeText, 10)
			if err != nil {
				log.WithCtx(ctx).Errorw("categorization", "provider", llmInfo, "md5", i.MD5, "content", i.Content, "error", err)
				return
			}
			if category == "低质量" {
				category = "low_quality"
			}
			runes := []rune(category)
			if len(runes) > 10 {
				count++
				if count > 3 {
					log.WithCtx(ctx).Errorw("categorization", "md5", i.MD5, "content", i.Content, "category", category, "error", "too many retries")
					category = "其他"
					return
				}
				goto Retry
			}
			state.Categorization[i] = category
			log.WithCtx(ctx).Debugw("categorization", "provider", llmInfo, "md5", i.MD5, "content", i.Content, "category", category)
		}(item)
	}

	wg.Wait()
	return state, nil // 始终返回 nil，单个失败不阻塞整体流程
}
