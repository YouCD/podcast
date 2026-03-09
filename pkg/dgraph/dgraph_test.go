package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"podcast/config"

	"podcast/internal/database/models"

	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
)

func init() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)
	//models.Init()
	//// 初始化Redis客户端
	//if err := pkg.InitRedisClient(); err != nil {
	//	log.Errorf("警告: Redis初始化失败: %v", err)
	//}
}

func TestDgraph_Insert(t *testing.T) {
	var resuklt []models.RssContent
	err := models.GetDb().Model(&models.RssContent{}).Where("dgraph != ''").Find(&resuklt).Error
	if err != nil {
		t.Errorf(" error = %v", err)
		return
	}
	d, err := New()
	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}
	//err = d.QueryOne(context.Background(), "浙江阿里巴巴云计算有限公司")
	//if err != nil {
	//	t.Errorf("QueryOne() error = %v", err)
	//	return
	//}
	for _, content := range resuklt {
		var payload types.DgraphPayload

		err = json.Unmarshal([]byte(content.Dgraph), &payload)
		if err != nil {
			t.Errorf("json.Unmarshal() error = %v", err)
			continue
		}
		if len(payload.Set) == 0 {
			continue
		}
		//for _, item := range payload.Set {
		//	if item.DgraphType == "" {
		//		fmt.Println(content.ID)
		//		continue
		//	}
		//}
		/*
			["xAI实验室","埃隆·马斯克的AI公司","xAI公司"]
		*/
		err = d.Insert(context.Background(), &payload)
		if err != nil {
			t.Errorf("Insert() error = %v", err)
			continue
		}
	}
}

func TestDgraph_QueryAliases(t *testing.T) {
	d, err := New()
	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}
	a, err := d.QueryAliases(context.Background(), "xAI实验室")
	if err != nil {
		t.Errorf("QueryAliases() error = %v, ", err)
	}
	fmt.Println(a)
}
