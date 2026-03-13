package milvus

import (
	"context"
	"fmt"
	"time"

	"podcast/pkg/types"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/spf13/cast"
	"github.com/youcd/toolkit/log"
)

type Milvus struct {
	c         *milvusclient.Client
	score     float32
	dimension int64
}

func (m *Milvus) Close(ctx context.Context) {
	err := m.c.Close(ctx)
	if err != nil {
		log.WithCtx(ctx).Error("关闭 Milvus 客户端时发生错误:", err)
	}
}

// NewMilvus 创建 Milvus 客户端
func NewMilvus(ctx context.Context, cfg *types.Milvus) *Milvus {
	m := Milvus{
		score:     cfg.Score,
		dimension: cfg.Dimension,
	}
	mClient, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: cfg.Endpoint,
		APIKey:  cfg.APIKey,
		DBName:  cfg.DBName,
	})
	if err != nil {
		log.WithCtx(ctx).Panic("连接失败:", err)
	}
	m.c = mClient

	return &m
}

func (m *Milvus) DedupSearch(ctx context.Context, queryVector []float32, collName string) (bool, string, float32, error) {
	endT := time.Now()
	startT := endT.Add(-time.Hour * 24)
	resultSet, err := m.c.Search(ctx, milvusclient.NewSearchOption(
		collName,
		1,
		[]entity.Vector{entity.FloatVector(queryVector)}).
		WithOutputFields("title").
		WithFilter(fmt.Sprintf("date >= %d and date <= %d", startT.Unix(), endT.Unix())),
	)
	if err != nil {
		return false, "", 0, fmt.Errorf("搜索失败: %w", err)
	}

	for _, item := range resultSet {
		if len(item.Scores) > 0 {
			if item.Scores[0] > m.score {
				asString, err := item.GetColumn("title").GetAsString(0)
				if err != nil {
					return true, "", item.Scores[0], nil
				}
				return true, asString, item.Scores[0], nil
			}
		}
	}
	return false, "", 0, nil
}

func (m *Milvus) Query24HData(ctx context.Context, collName string) (map[string][]float32, error) {
	endT := time.Now()
	startT := endT.Add(-time.Hour * 24)
	expr := fmt.Sprintf("date >= %d and date <= %d", startT.Unix(), endT.Unix())

	resultSet, err := m.c.Query(ctx, milvusclient.NewQueryOption(collName).
		WithOutputFields("title", "vector").
		WithFilter(expr),
	)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}
	result := make(map[string][]float32)
	for index := range resultSet.ResultCount {
		title, err := resultSet.GetColumn("title").GetAsString(index)
		if err != nil {
			continue
		}
		vector, err := resultSet.GetColumn("vector").Get(index)
		if err != nil {
			continue
		}
		if floatVector, ok := vector.(entity.FloatVector); ok {
			result[title] = floatVector
		}
	}

	return result, err
}

func (m *Milvus) CreateDedupCollection(ctx context.Context, collName string) error {
	schema := &entity.Schema{
		Description: "title去重",
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},

			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": cast.ToString(m.dimension), // 默认维度
				},
			},
			{
				Name:     "title",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "255",
				},
			},
			{
				Name:     "date",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "md5",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "32",
				},
			},
		},
		EnableDynamicField: true,
	}

	option := milvusclient.NewCreateCollectionOption(collName, schema)
	err := m.c.CreateCollection(
		ctx, // ctx
		option,
	)
	if err != nil {
		return fmt.Errorf("创建集合失败: %w", err)
	}

	op := milvusclient.NewCreateIndexOption(collName, "vector", index.NewFlatIndex(entity.COSINE))
	_, err = m.c.CreateIndex(ctx, op)
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	_, err = m.c.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collName))
	if err != nil {
		log.WithCtx(ctx).Errorf("加载集合失败: %v\n", err)
		return err
	}
	return nil
}

func (m *Milvus) Insert(ctx context.Context, date time.Time, md5, title string, vectors []float32, collName string) error {
	option := milvusclient.NewRowBasedInsertOption(collName).
		WithFloatVectorColumn("vector", int(m.dimension), [][]float32{vectors}).
		WithVarcharColumn("title", []string{title}).
		WithVarcharColumn("md5", []string{md5}).
		WithInt64Column("date", []int64{date.Unix()})

	// 插入到Milvus集合
	insertResult, err := m.c.Insert(ctx, option)
	if err != nil {
		return fmt.Errorf("插入数据失败: %w", err)
	}
	log.WithCtx(ctx).Infow("milvus", "insertResult", insertResult.IDs, "title", title, "md5", md5)
	return nil
}
