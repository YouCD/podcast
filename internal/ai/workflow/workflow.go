package workflow

import (
	"context"
	"fmt"
	"time"

	"podcast/internal/ai/milvus"

	"podcast/pkg/types"

	"github.com/cloudwego/eino/compose"
	"github.com/youcd/toolkit/log"
)

// graphState 全局状态（用于节点间共享数据）
type graphState struct {
	RawItems       []*types.RSSItem
	Filtered       []*types.RSSItem
	UniqueItems    []*types.RSSItem
	Categorization map[*types.RSSItem]string
	LlmResult      []*types.RSSItem
	Errors         []error
}

func buildRSSWorkflow(cfg *types.RagConfig) *compose.Graph[[]*types.RSSSource, []*types.RSSItem] {
	graph := compose.NewGraph[[]*types.RSSSource, []*types.RSSItem]()
	// 节点 1: RSS 获取
	_ = graph.AddLambdaNode("fetch_rss", compose.InvokableLambda(fetchFeeds))

	// 节点 2: 日期过滤
	_ = graph.AddLambdaNode("filter_today", compose.InvokableLambda(filterToday))

	// 节点 3: 去重（使用闭包注入配置）
	_ = graph.AddLambdaNode("deduplicate", compose.InvokableLambda(func(ctx context.Context, state *graphState) (*graphState, error) {
		return deduplicate(ctx, state, cfg)
	}))

	// 节点 4: 并行分类处理
	_ = graph.AddLambdaNode("categorization", compose.InvokableLambda(categorization))

	// 节点 5: rss内容分析
	_ = graph.AddLambdaNode("analyze_rss", compose.InvokableLambda(analyzeRss))

	// 节点 7: 关系分析
	_ = graph.AddLambdaNode("dgraph", compose.InvokableLambda(dgraph))

	// 节点 8: 保存
	_ = graph.AddLambdaNode("save", compose.InvokableLambda(func(ctx context.Context, rssItems []*types.RSSItem) ([]*types.RSSItem, error) {
		return save(ctx, rssItems, cfg)
	}))

	// 编排
	_ = graph.AddEdge(compose.START, "fetch_rss")
	_ = graph.AddEdge("fetch_rss", "filter_today")
	_ = graph.AddEdge("filter_today", "deduplicate")
	_ = graph.AddEdge("deduplicate", "categorization")
	_ = graph.AddEdge("categorization", "analyze_rss")
	_ = graph.AddEdge("analyze_rss", "dgraph")
	_ = graph.AddEdge("dgraph", "save")
	_ = graph.AddEdge("save", compose.END)
	return graph
}

func New(ctx context.Context, cfg *types.RagConfig) (compose.Runnable[[]*types.RSSSource, []*types.RSSItem], error) {
	// 构建工作流（注入配置）
	workflow := buildRSSWorkflow(cfg)
	// 编译（启用流式处理和回调）
	runnable, err := workflow.Compile(ctx)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, fmt.Errorf("编译工作流失败：%w", err)
	}
	m := milvus.NewMilvus(ctx, cfg.Milvus)
	defer m.Close(ctx)
	data, err := m.Query24HData(ctx, cfg.Milvus.DedupCollection)
	if err == nil {
		for title, value := range data {
			cacheStore.Set(title, value, time.Hour*24)
		}
	}
	go func() {
		clearTicker := time.NewTicker(time.Hour)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		defer clearTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.WithCtx(ctx).Info("cache goroutine stopped")
				return
			case <-ticker.C:
				log.WithCtx(ctx).Infof("cache size: %d", cacheStore.ItemCount())
			case <-clearTicker.C:
				cacheStore.DeleteExpired()
				log.WithCtx(ctx).Infof("cache size after cleanup: %d", cacheStore.ItemCount())
			}
		}
	}()

	return runnable, nil
}
