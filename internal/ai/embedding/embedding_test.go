package embedding

import (
	"context"
	"fmt"
	"testing"

	"podcast/config"
	"podcast/internal/ai/milvus"
)

func init() {
	config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	// models.Init()
}

func Test_embedding(t *testing.T) {
	//Title := "叮咚买菜被美团收购， 即时零售进入巨头争战期"
	//embedding, err := New(context.Background(), Title)
	//if err != nil {
	//	t.Error(err)
	//}
	//t.Log(embedding)

	m := milvus.New(context.Background())

	dedup, err := m.Query(context.Background())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(dedup)
	//err = m.Insert(context.Background(), time.Now(), "item.MD5", Title, embedding[0].New)
	//if err != nil {
	//	t.Error(err)
	//}
	//d := dao.NewRssContentDao()
	//
	//dateRange, err2 := d.FindByDateRange(context.Background(), time.Now().Add(-time.Hour*24), time.Now())
	//if err2 != nil {
	//	t.Error(err2)
	//}
	//
	//for _, item := range dateRange {
	//	fmt.Println(item.Title)
	//	embedding, err := New(context.Background(), item.Title)
	//	if err != nil {
	//		t.Error(err)
	//	}
	//	t.Log(embedding)
	//
	//	m := milvus.New(context.Background())
	//
	//	err = m.Insert(context.Background(), item.Date, item.MD5, item.Title, embedding[0].New)
	//	if err != nil {
	//		t.Error(err)
	//	}
	//}
}
