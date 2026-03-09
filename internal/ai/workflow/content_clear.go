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

func contentClear(ctx context.Context, state *graphState) ([]*types.RSSItem, error) {
	if len(state.LlmResult) == 0 {
		return nil, ErrNotLLMResult
	}
	const maxConcurrency = 2
	sem := semaphore.NewWeighted(maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range state.LlmResult {
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

			msgs, err := newCleanTemplate().Format(ctx, map[string]any{
				"content": i.Content,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("content_clear", "error", err)
				return
			}
			llmResult, llmInfo, err := common.RunModelGenerate(ctx, "content_clear", msgs, openai.ChatCompletionResponseFormatTypeText, 5)
			if err != nil {
				log.WithCtx(ctx).Errorw("content_clear", "provider", llmInfo, "md5", i.MD5, "content", i.Content, "error", err)
				return
			}
			i.Content = llmResult
		}(item)
	}
	wg.Wait()

	return state.LlmResult, nil // 始终返回 nil，单个失败不阻塞整体流程
}
