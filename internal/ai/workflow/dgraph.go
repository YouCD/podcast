package workflow

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
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

	log.WithCtx(ctx).Infof("[阶段6/7] 开始关系分析，共 %d 条", len(state.LlmResult))

	var temp []*types.RSSItem
	var mu sync.Mutex

	// 创建信号量控制并发
	sem := semaphore.NewWeighted(state.maxConcurrency)
	var wg sync.WaitGroup

	for _, item := range state.LlmResult {
		if !analyzeDgraph(item.Categories) {
			temp = append(temp, item)
			continue
		}

		wg.Add(1)
		// 获取信号量
		if err := sem.Acquire(ctx, 1); err != nil {
			log.WithCtx(ctx).Errorw("dgraph: acquire semaphore failed", "error", err)
			wg.Done()
			continue
		}

		go func(item *types.RSSItem) {
			defer wg.Done()
			defer sem.Release(1)

			log.WithCtx(ctx).Debugw("[阶段6/7] 开始关系分析", "md5", item.MD5, "title", item.Title, "category", item.Categories)

			msgs, err := newRssDgraphTemplate(item.Categories).Format(ctx, map[string]any{
				"date":    item.Date,
				"content": item.Content,
				"title":   item.Title,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "error", err)
				return
			}
			llmResult, llmInfo, err := common.RunModelGenerate(ctx, state.llmPool, "dgraph", msgs, openai.ChatCompletionResponseFormatTypeJSONObject, 5, state.llmTimeout)
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "provider", llmInfo, "md5", item.MD5, "content", item.Content, "error", err)
				return
			}
			var payload types.DgraphPayload
			cleanResult := stripJSONFence(llmResult)
			err = json.Unmarshal([]byte(cleanResult), &payload)
			if err != nil {
				log.WithCtx(ctx).Errorw("dgraph", "llmResult", llmResult, "error", err)
				return
			}
			item.Dgraph = cleanResult
			mu.Lock()
			temp = append(temp, item)
			mu.Unlock()
		}(item)
	}

	// 等待所有任务完成
	wg.Wait()

	log.WithCtx(ctx).Infof("[阶段6/7] 关系分析完成，共 %d 条", len(temp))
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

// stripJSONFence 剥离 LLM 输出中可能包裹的 markdown 代码围栏（```json ... ```）
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
