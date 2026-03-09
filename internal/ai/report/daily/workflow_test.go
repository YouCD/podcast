package daily

import (
	"context"
	"fmt"
	"podcast/config"
	"podcast/internal/database/models"
	"testing"

	"github.com/youcd/toolkit/log"
)

func init() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)
	models.Init()
	//// 初始化Redis客户端
	//if err := pkg.InitRedisClient(); err != nil {
	//	log.Errorf("警告: Redis初始化失败: %v", err)
	//}
}

func TestNew(t *testing.T) {
	c, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.Invoke(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(data.report)
}
