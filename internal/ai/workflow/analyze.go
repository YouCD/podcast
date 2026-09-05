package workflow

import (
	"context"
	"encoding/json"
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

	log.WithCtx(ctx).Infof("[阶段5/7] 开始内容分析，共 %d 条", len(state.Categorization))

	var rssItems []*types.RSSItem
	for item, s := range state.Categorization {
		item.Categories = s
		rssItems = append(rssItems, item)
	}

	// 创建信号量控制并发
	sem := semaphore.NewWeighted(state.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range rssItems {
		wg.Add(1)
		// 获取信号量
		if err := sem.Acquire(ctx, 1); err != nil {
			log.WithCtx(ctx).Errorw("analyze_rss: acquire semaphore failed", "error", err)
			wg.Done()
			continue
		}

		go func(item *types.RSSItem) {
			defer wg.Done()
			defer sem.Release(1)

			msgs, err := newRssAnalyzeTemplate(ctx, item.Categories).Format(ctx, map[string]any{
				"content": item.Content,
				"date":    time.Now(),
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("analyze_rss", "error", err)
				return
			}
			var llmResult string
			var llmInfo *types.LLMInfo
			var retryCount int
			const maxRetry = 3
		Retry:
			llmResult, llmInfo, err = common.RunModelGenerate(ctx, state.llmPool, "analyze_rss", msgs, openai.ChatCompletionResponseFormatTypeJSONObject, 5, state.llmTimeout)
			if err != nil {
				retryCount++
				if retryCount < maxRetry {
					log.WithCtx(ctx).Warnw("analyze_rss retry", "provider", llmInfo, "md5", item.MD5, "attempt", retryCount, "error", err)
					goto Retry
				}
				log.WithCtx(ctx).Errorw("analyze_rss failed after retries", "provider", llmInfo, "md5", item.MD5, "error", err)
				return
			}

			var result types.LLMResult
			err = json.Unmarshal([]byte(stripJSONFence(llmResult)), &result)
			if err != nil {
				log.WithCtx(ctx).Errorw("analyze_rss", "provider", llmInfo, "title", item.Title, "md5", item.MD5, "error", err)
				return
			}
			marshal, _ := json.Marshal(result)
			item.LLMResult = string(marshal)
			log.WithCtx(ctx).Debugw("analyzeRss", "provider", llmInfo, "md5", item.MD5, "content", item.LLMResult)
		}(item)
	}

	// 等待所有任务完成
	wg.Wait()

	// 收集有结果的项目
	for _, item := range rssItems {
		if item.LLMResult != "" {
			mu.Lock()
			state.LlmResult = append(state.LlmResult, item)
			mu.Unlock()
		}
	}
	log.WithCtx(ctx).Infof("[阶段5/7] 分析完成，共 %d 条", len(state.LlmResult))
	return state, nil // 始终返回 nil，单个失败不阻塞整体流程
}
