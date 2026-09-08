package dao

import (
	"context"
	"testing"

	"github.com/youcd/toolkit/log"
)

func TestKeyInfoDao_FindByKeynameAndGenre(t *testing.T) {
	dao := &keyInfoDao{}

	// 测试用例1: 查找存在的记录
	gotKeyInfo, err := dao.FindByKeynameAndGenre(context.Background(), "test_key", 1)
	if err != nil {
		log.WithCtx(context.Background()).Errorf("查找KeyInfo错误: %v", err)
	} else {
		log.WithCtx(context.Background()).Infof("找到KeyInfo: %#v", gotKeyInfo)
	}

	// 测试用例2: 查找不存在的记录
	gotKeyInfo2, err := dao.FindByKeynameAndGenre(context.Background(), "nonexistent_key", 999)
	if err != nil {
		log.WithCtx(context.Background()).Errorf("查找不存在的KeyInfo错误（预期）: %v", err)
	} else {
		log.WithCtx(context.Background()).Infof("找到KeyInfo: %#v", gotKeyInfo2)
	}
}
