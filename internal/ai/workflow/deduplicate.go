package workflow

import (
	"context"
	"time"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/pgvector"
	"podcast/internal/database/dao"
	"podcast/pkg/types"

	"github.com/patrickmn/go-cache"
	"github.com/youcd/toolkit/log"
)

// cacheStore 缓存存储，用于向量去重
var cacheStore = cache.New(24*time.Hour, 24*time.Hour)

// Deduplicator 去重处理器，使用依赖注入
type Deduplicator struct {
	rssDao   dao.RssContentDao
	embedder *embedding.Embedder
	pgVector *pgvector.PgVector
	cfg      *types.RagConfig
}

// NewDeduplicator 创建去重处理器
func NewDeduplicator(rssDao dao.RssContentDao, embedder *embedding.Embedder, pgVectorClient *pgvector.PgVector, cfg *types.RagConfig) *Deduplicator {
	return &Deduplicator{
		rssDao:   rssDao,
		embedder: embedder,
		pgVector: pgVectorClient,
		cfg:      cfg,
	}
}

// deduplicate 去重处理（使用依赖注入）
func (d *Deduplicator) deduplicate(ctx context.Context, state *graphState) (*graphState, error) {
	log.WithCtx(ctx).Info("[阶段3/7] 开始去重处理")
	output := []*types.RSSItem{}

	for _, item := range state.Filtered {
		// 查询数据库中是否已存在该键
		existingPost, err := d.rssDao.FindByMD5(ctx, item.MD5)
		if err != nil {
			log.WithCtx(ctx).Errorw("deduplicate", "MD5", item.MD5, "err", err)
			continue
		}
		var need bool
		if len(existingPost) == 0 {
			need = true
		}
		if len(existingPost) > 0 {
			// 如果已经存在内容，且没有LLMResult，则发送到下一阶段
			if existingPost[0].Categories != "low_quality" && existingPost[0].LLMResult == "" {
				need = true
			}
			// 若已经存在的内容，打印提示
			log.WithCtx(ctx).Debugw("deduplicate_existing", "MD5", item.MD5)
		}

		// 添加到结果中
		if need {
			output = append(output, item)
			log.WithCtx(ctx).Debugw("deduplicate_need_embedding", "MD5", item.MD5)
		}
	}
	log.WithCtx(ctx).Infow("deduplicate_need_embedding_count", "in_mysql", len(output))
	// 向量去重处理
	dedup := d.embeddingDedup(ctx, output)
	log.WithCtx(ctx).Infow("deduplicate_embedding_count", "in_embedding", len(dedup))

	state.UniqueItems = dedup
	log.WithCtx(ctx).Infof("[阶段3/7] 去重完成，共 %d 条", len(dedup))
	return state, nil
}

func (d *Deduplicator) embeddingDedup(ctx context.Context, rss []*types.RSSItem) []*types.RSSItem {
	var items []*types.RSSItem
	for _, item := range rss {
		var vector []float32
		_, b := cacheStore.Get(item.Title)
		if b {
			log.WithCtx(ctx).Debugw("embedding_dedup", "MD5", item.MD5, "title", item.Title, "cache", "hit")
			continue
		} else {
			vectorData, err := d.embedder.CreateEmbeddings(ctx, item.Title)
			if err != nil {
				log.WithCtx(ctx).Errorw("embedding_create", "MD5", item.MD5, "title", item.Title, "err", err)
				continue
			}
			vector = vectorData
		}

		dedup, title, score, err := d.pgVector.DedupSearch(ctx, vector, d.cfg.PgVector.DedupCollection)
		if err != nil {
			log.WithCtx(ctx).Errorw("dedup", "err", err)
			continue
		}
		if dedup {
			log.WithCtx(ctx).Debugw("embedding_dedup", "MD5", item.MD5, "embedding", "hit", "emb_title", title, "src_title", item.Title, "score", score)
		} else {
			item.Vector = vector // 保存向量到 RSSItem，供后续使用
			items = append(items, item)
			log.WithCtx(ctx).Debugw("embedding_dedup_pass", "MD5", item.MD5, "title", item.Title)
			continue
		}
	}
	return items
}
