package workflow

import (
	"context"
	"podcast/config"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"time"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/milvus"
	"podcast/pkg/types"

	"github.com/patrickmn/go-cache"
	"github.com/youcd/toolkit/log"
)

// deduplicate 去重处理
func deduplicate(ctx context.Context, state *graphState) (*graphState, error) {
	log.WithCtx(ctx).Info("开始去重处理")
	output := []*types.RSSItem{}
	var rssContentDao = dao.NewRssContentDao(models.GetDb())
	for _, item := range state.Filtered {
		// 查询数据库中是否已存在该键
		existingPost, err := rssContentDao.FindByMD5(ctx, item.MD5)
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
	dedup := embeddingDedup(ctx, output...)
	log.WithCtx(ctx).Infow("deduplicate_embedding_count", "in_embedding", len(dedup))

	state.UniqueItems = dedup
	return state, nil
}

var cacheStore = cache.New(24*time.Hour, 24*time.Hour)

func embeddingDedup(ctx context.Context, rss ...*types.RSSItem) []*types.RSSItem {
	var items []*types.RSSItem
	m := milvus.New(ctx)
	defer m.Close(ctx)
	for _, item := range rss {
		var vector []float32
		_, b := cacheStore.Get(item.Title)
		if b {
			log.WithCtx(ctx).Debugw("embedding_dedup", "MD5", item.MD5, "title", item.Title, "cache", "hit")
			continue
		} else {
			emb, err := embedding.New(ctx, item.Title)
			if err != nil {
				log.WithCtx(ctx).Errorw("embedding_create", "MD5", item.MD5, "title", item.Title, "err", err)
				continue
			}
			vector = emb[0].Embedding
			cacheStore.Set(item.Title, vector, 24*time.Hour)
		}

		dedup, title, score, err := m.DedupSearch(ctx, vector, config.Cfg.Database.Milvus.DedupCollection)
		if err != nil {
			log.WithCtx(ctx).Errorw("dedup", "err", err)
			continue
		}
		if dedup {
			log.WithCtx(ctx).Debugw("embedding_dedup", "MD5", item.MD5, "embedding", "hit", "emb_title", title, "src_title", item.Title, "score", score)
		} else {
			items = append(items, item)
			err = m.Insert(ctx, item.Date, item.MD5, item.Title, vector, config.Cfg.Database.Milvus.DedupCollection)
			if err != nil {
				log.WithCtx(ctx).Errorw("embedding_dedup", "MD5", item.MD5, "title", item.Title, "err", err)
			} else {
				log.WithCtx(ctx).Debugw("embedding_dedup_insert", "MD5", item.MD5, "title", item.Title)
			}
			continue
		}
	}
	return items
}
