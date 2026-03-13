package workflow

import (
	"context"

	"podcast/config"
	"podcast/internal/database/models"

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
