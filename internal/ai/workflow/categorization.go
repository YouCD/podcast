package workflow

import (
	"context"
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

	log.WithCtx(ctx).Infof("[阶段4/7] 开始分类处理，共 %d 条", len(state.UniqueItems))

	// 创建信号量控制并发
	sem := semaphore.NewWeighted(state.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range state.UniqueItems {
		wg.Add(1)
		// 获取信号量
		if err := sem.Acquire(ctx, 1); err != nil {
			log.WithCtx(ctx).Errorw("categorization: acquire semaphore failed", "error", err)
			wg.Done()
			continue
		}

		go func(item *types.RSSItem) {
			defer wg.Done()
			defer sem.Release(1)

			log.WithCtx(ctx).Debugw("[阶段4/7] 开始分类", "md5", item.MD5, "title", item.Title, "source", item.Source)

			msgs, err := newCategorizationTemplate(ctx).Format(ctx, map[string]any{
				"title":   item.Title,
				"content": item.Content,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("categorization", "md5", item.MD5, "content", item.Content, "error", err)
				mu.Lock()
				state.Categorization[item] = "其他"
				mu.Unlock()
				return
			}
			var category string
			var llmInfo *types.LLMInfo
			var retryCount int
			const maxRetry = 3
		Retry:
			category, llmInfo, err = common.RunModelGenerate(ctx, state.llmPool, "categorization", msgs, openai.ChatCompletionResponseFormatTypeText, 3, state.llmTimeout)
			if err != nil {
				retryCount++
				if retryCount < maxRetry {
					log.WithCtx(ctx).Warnw("categorization retry", "provider", llmInfo, "md5", item.MD5, "attempt", retryCount, "error", err)
					goto Retry
				}
				log.WithCtx(ctx).Errorw("categorization failed after retries", "provider", llmInfo, "md5", item.MD5, "content", item.Content, "error", err)
				mu.Lock()
				state.Categorization[item] = "其他"
				mu.Unlock()
				return
			}
			if category == "低质量" {
				category = "low_quality"
			}
			runes := []rune(category)
			if category != "low_quality" && len(runes) > 10 {
				retryCount++
				if retryCount < maxRetry {
					log.WithCtx(ctx).Warnw("categorization category too long, retrying", "md5", item.MD5, "category", category, "attempt", retryCount, "category", category)
					goto Retry
				}
				log.WithCtx(ctx).Errorw("categorization", "md5", item.MD5, "content", item.Content, "category", category, "error", "too many retries")
				category = "其他"
			}
			mu.Lock()
			state.Categorization[item] = category
			mu.Unlock()
			log.WithCtx(ctx).Debugw("categorization", "provider", llmInfo, "md5", item.MD5, "content", item.Content, "category", category)
		}(item)
	}

	// 等待所有任务完成
	wg.Wait()

	log.WithCtx(ctx).Infof("[阶段4/7] 分类完成，共 %d 条", len(state.Categorization))
	return state, nil // 始终返回 nil，单个失败不阻塞整体流程
}
