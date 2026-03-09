package workflow

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	dgraphC "podcast/pkg/dgraph"
	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func save(ctx context.Context, rssItems []*types.RSSItem) ([]*types.RSSItem, error) {
	log.WithCtx(ctx).Info("开始保存到数据库")
	var milvusData []*models.RssContent
	var dbData []*models.RssContent
	var rssContentDao = dao.NewRssContentDao(models.GetDb())
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
			// 创建新记录
			post := models.RssContent{
				Title:      item.Title,
				Content:    item.Content,
				Date:       item.Date,
				Categories: item.Categories,
				Source:     item.Source,
				Link:       item.Link,
				MD5:        item.MD5,
				LLMResult:  item.LLMResult,
				Dgraph:     item.Dgraph,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			dbData = append(dbData, &post)

			if item.Categories != "low_quality" {
				milvusData = append(milvusData, &post)
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
	err := rssContentDao.BatchCreate(ctx, milvusData...)
	if err != nil {
		log.WithCtx(ctx).Error("保存RSS项目到数据库失败: %v", err)
	} else {
		log.WithCtx(ctx).Info("数据库保存完成")
	}

	wg := sync.WaitGroup{}
	for _, item := range milvusData {
		wg.Add(1) // 每个 goroutine 启动前增加计数
		go func(item *models.RssContent) {
			defer wg.Done() // 每个 goroutine 结束后减少计数
			err := saveToMilvus(ctx, item)
			if err != nil {
				log.WithCtx(ctx).Errorf("保存RSS项目到 Milvus,item :%v  err: %v", item, err)
			}
		}(item)
	}
	wg.Wait() // 等待所有 goroutine 完成

	d, err := dgraphC.New()
	if err != nil {
		log.WithCtx(ctx).Error("保存RSS项目到 Milvus,err: %v", err)
		return rssItems, nil
	}
	defer d.Close()

	wgA := sync.WaitGroup{}
	for _, item := range milvusData {
		var payload types.DgraphPayload
		err = json.Unmarshal([]byte(item.Dgraph), &payload)
		if err != nil {
			log.WithCtx(ctx).Errorf("解析Dgraph数据失败: %v", err)
			continue
		}
		if len(payload.Set) == 0 {
			continue
		}

		wgA.Add(1)
		go func(p *types.DgraphPayload) {
			defer wgA.Done()
			err := d.Insert(ctx, p)
			if err != nil {
				log.WithCtx(ctx).Errorf("保存 Dgraph err: %v", err)
			}
		}(&payload)
		wgA.Wait()
	}
	return rssItems, nil
}
