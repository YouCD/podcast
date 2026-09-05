package workflow

import (
	"context"
	"encoding/json"
	"time"

	"podcast/internal/ai/llm"
	"podcast/internal/ai/pgvector"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	dgraphC "podcast/pkg/dgraph"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func save(ctx context.Context, rssItems []*types.RSSItem, cfg *types.RagConfig, llmPool *llm.LLMPool, pgVector *pgvector.PgVector) ([]*types.RSSItem, error) {
	log.WithCtx(ctx).Infof("[阶段7/7] 开始保存到数据库，共 %d 条", len(rssItems))
	var pgvectorData []*models.RssContent
	var dbData []*models.RssContent
	rssContentDao := dao.NewRssContentDao(models.GetDb())
	for _, item := range rssItems {
		// 检查是否已存在相同MD5的内容
		existingPost, err := rssContentDao.FindByMD5(ctx, item.MD5)
		if err != nil {
			log.WithCtx(ctx).Errorf("查询数据库失败: %v", err)
			continue
		}
		//nolint:all
		if len(existingPost) == 0 {
			if item.Date.IsZero() {
				item.Date = time.Now()
			}
			// 创建新记录（截断超出数据库列长度的字段，避免整批写入失败）
			post := models.RssContent{
				Title:      truncateRunes(item.Title, 255),
				Content:    item.Content,
				Date:       item.Date,
				Categories: truncateRunes(item.Categories, 100),
				Source:     truncateRunes(item.Source, 100),
				Link:       truncateRunes(item.Link, 255),
				MD5:        item.MD5,
				LLMResult:  item.LLMResult,
				Dgraph:     item.Dgraph,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			dbData = append(dbData, &post)

			if item.Categories != "low_quality" {
				pgvectorData = append(pgvectorData, &post)
			}
		}
		if len(existingPost) == 1 {
			if existingPost[0].LLMResult == "" {
				// 更新已存在的记录（仅当LLMResult为空时）
				//nolint:modernize
				dataMap := map[string]interface{}{
					"llm_result": item.LLMResult,
					"categories": item.Categories,
				}
				if err = rssContentDao.UpdateByFields(ctx, dao.FieldKv{Field: "id", Value: existingPost[0].ID}, dataMap); err != nil {
					log.WithCtx(ctx).Error("更新RSS项目到数据库失败: %v", err)
				}
			}
		}
	}
	err := rssContentDao.BatchCreate(ctx, dbData...)
	if err != nil {
		log.WithCtx(ctx).Errorf("保存RSS项目到数据库失败: %v", err)
	} else {
		log.WithCtx(ctx).Info("数据库保存完成")
		// 数据库保存成功后，写入缓存和DedupCollection
		for _, item := range rssItems {
			if len(item.Vector) > 0 {
				cacheStore.Set(item.Title, item.Vector, 24*time.Hour)
				log.WithCtx(ctx).Debugw("cache_set after save", "title", item.Title)
				// 插入到DedupCollection用于语义去重
				err := pgVector.Insert(ctx, item.Date, item.MD5, item.Title, item.Vector, cfg.PgVector.DedupCollection)
				if err != nil {
					log.WithCtx(ctx).Errorw("dedup_collection_insert failed", "title", item.Title, "error", err)
				} else {
					log.WithCtx(ctx).Debugw("dedup_collection_insert success", "title", item.Title)
				}
			}
		}
	}

	for _, item := range pgvectorData {
		err := saveToPgVector(ctx, llmPool, item, cfg)
		if err != nil {
			log.WithCtx(ctx).Errorf("保存RSS项目到 PgVector,item :%v  err: %v", item, err)
		}
	}

	d, err := dgraphC.New(cfg.DgraphHost)
	if err != nil {
		log.WithCtx(ctx).Error("保存RSS项目到 PgVector,err: %v", err)
		return rssItems, nil
	}
	defer d.Close()

	for _, item := range pgvectorData {
		var payload types.DgraphPayload
		err = json.Unmarshal([]byte(item.Dgraph), &payload)
		if err != nil {
			log.WithCtx(ctx).Errorf("解析Dgraph数据失败: %v", err)
			continue
		}
		if len(payload.Set) == 0 {
			continue
		}

		err := d.Insert(ctx, &payload)
		if err != nil {
			log.WithCtx(ctx).Errorf("保存 Dgraph err: %v", err)
		}
	}
	log.WithCtx(ctx).Info("[阶段7/7] 保存完成")
	return rssItems, nil
}

// truncateRunes 将字符串截断为最多 maxRunes 个字符（按 rune 计算，避免截断多字节中文）
func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}
