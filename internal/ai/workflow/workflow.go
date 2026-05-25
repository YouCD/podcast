package workflow

import (
	"context"
	"fmt"
	"time"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/milvus"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"

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
	llmPool        *llm.LLMPool
}

// RSSWorkflow RSS工作流，包含依赖注入的组件
type RSSWorkflow struct {
	cfg          *types.RagConfig
	deduplicator *Deduplicator
	milvusClient *milvus.Milvus
	llmPool      *llm.LLMPool
}

// NewRSSWorkflow 创建RSS工作流实例
func NewRSSWorkflow(ctx context.Context, cfg *types.RagConfig, llmPool *llm.LLMPool) (*RSSWorkflow, error) {
	// 初始化 milvus 客户端
	milvusClient := milvus.NewMilvus(ctx, cfg.Milvus)

	// 初始化 embedder
	embedder, err := embedding.NewEmbedder(ctx, cfg.Embedding)
	if err != nil {
		milvusClient.Close(ctx)
		return nil, fmt.Errorf("创建 embedder 失败: %w", err)
	}

	// 初始化 DAO
	rssDao := dao.NewRssContentDao(models.GetDb())

	// 创建去重器
	deduplicator := NewDeduplicator(rssDao, embedder, milvusClient, cfg)

	// 预热缓存
	data, err := milvusClient.Query24HData(ctx, cfg.Milvus.DedupCollection)
	if err == nil {
		for title, value := range data {
			cacheStore.Set(title, value, time.Hour*24)
		}
	}

	return &RSSWorkflow{
		cfg:          cfg,
		deduplicator: deduplicator,
		milvusClient: milvusClient,
		llmPool:      llmPool,
	}, nil
}

// buildGraph 构建工作流图
func (w *RSSWorkflow) buildGraph() *compose.Graph[[]*types.RSSSource, []*types.RSSItem] {
	graph := compose.NewGraph[[]*types.RSSSource, []*types.RSSItem]()

	// 节点 1: RSS 获取
	_ = graph.AddLambdaNode("fetch_rss", compose.InvokableLambda(fetchFeeds))

	// 节点 2: 日期过滤
	_ = graph.AddLambdaNode("filter_today", compose.InvokableLambda(filterToday))

	// 节点 3: 去重（使用依赖注入的 Deduplicator）
	_ = graph.AddLambdaNode("deduplicate", compose.InvokableLambda(func(ctx context.Context, state *graphState) (*graphState, error) {
		return w.deduplicator.deduplicate(ctx, state)
	}))

	// 节点 4: 并行分类处理
	_ = graph.AddLambdaNode("categorization", compose.InvokableLambda(func(ctx context.Context, state *graphState) (*graphState, error) {
		state.llmPool = w.llmPool
		return categorization(ctx, state)
	}))

	// 节点 5: rss内容分析
	_ = graph.AddLambdaNode("analyze_rss", compose.InvokableLambda(func(ctx context.Context, state *graphState) (*graphState, error) {
		state.llmPool = w.llmPool
		return analyzeRss(ctx, state)
	}))

	// 节点 7: 关系分析
	_ = graph.AddLambdaNode("dgraph", compose.InvokableLambda(func(ctx context.Context, state *graphState) ([]*types.RSSItem, error) {
		state.llmPool = w.llmPool
		return dgraph(ctx, state)
	}))

	// 节点 8: 保存
	_ = graph.AddLambdaNode("save", compose.InvokableLambda(func(ctx context.Context, rssItems []*types.RSSItem) ([]*types.RSSItem, error) {
		return save(ctx, rssItems, w.cfg, w.llmPool, w.milvusClient)
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

// Compile 编译工作流
func (w *RSSWorkflow) Compile(ctx context.Context) (compose.Runnable[[]*types.RSSSource, []*types.RSSItem], error) {
	workflow := w.buildGraph()
	runnable, err := workflow.Compile(ctx)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, fmt.Errorf("编译工作流失败：%w", err)
	}
	return runnable, nil
}

// Close 关闭工作流资源
func (w *RSSWorkflow) Close(ctx context.Context) {
	if w.milvusClient != nil {
		w.milvusClient.Close(ctx)
	}
}

// StartCacheManager 启动缓存管理 goroutine
func (w *RSSWorkflow) StartCacheManager(ctx context.Context) {
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
}

// New 创建工作流（兼容旧接口，推荐使用 NewRSSWorkflow）
// Deprecated: 请使用 NewRSSWorkflow
func New(ctx context.Context, cfg *types.RagConfig, llmPool *llm.LLMPool) (compose.Runnable[[]*types.RSSSource, []*types.RSSItem], error) {
	workflow, err := NewRSSWorkflow(ctx, cfg, llmPool)
	if err != nil {
		return nil, err
	}

	// 启动缓存管理
	workflow.StartCacheManager(ctx)

	// 编译工作流
	return workflow.Compile(ctx)
}
