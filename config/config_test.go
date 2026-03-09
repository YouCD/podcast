package config

import (
	"context"
	"testing"

	"github.com/youcd/toolkit/log"
)

func TestLoadAppConfig(t *testing.T) {
	// 初始化数据库连接
	c, err := LoadAppConfig("")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)
	for _, rss := range c.RSS {
		log.WithCtx(context.Background()).Infof("%#v", rss)
	}
}
