package dao

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"podcast/config"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/internal/database/models"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/youcd/toolkit/log"
)

func init() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	if err != nil {
		panic(err)
	}
	_, _ = models.Init(c)
	log.WithCtx(context.Background()).Infof("%#v", c)
}

func TestCommunityPostDao_FindByMD5(t *testing.T) {
	dao := NewRssContentDao(models.GetDb())
	//gotPost, err := dao.FindByMD5(context.Background(), "e90fa082a72794fcb54e079bd8d9b240")
	//if gotPost == nil {
	//	log.Errorf("错误: %v", err)
	//	return
	//}
	//log.Infof("%#v", gotPost)
	//if err != nil {
	//	log.Errorf("错误: %v", err)
	//}
	all, err := dao.FindAll(context.Background())
	if err != nil {
		log.WithCtx(context.Background()).Errorf("错误: %v", err)
	}
	var val int
	var vals []int
	for _, content := range all {
		lens := len([]rune(content.Content))
		if lens > val {
			val = lens
		}
		vals = append(vals, lens)
	}
	log.WithCtx(context.Background()).Infof("%d", val)
	slice.Sort(vals)
	log.WithCtx(context.Background()).Infof("%v", vals)

	var aa int
	for _, i := range vals {
		aa += i
	}
	fmt.Println("平均 ", aa/len(vals))
}

func Test_rssContentDao_FindByDateRange(t *testing.T) {
	dao := NewRssContentDao(models.GetDb())
	yesterday := time.Now().AddDate(0, 0, -1)
	startDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	endDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, yesterday.Location())
	dateRange, err := dao.FindByDateRange(context.Background(), startDate, endDate)
	if err != nil {
		log.WithCtx(context.Background()).Errorf("错误: %v", err)
	}
	var concatenatedContent strings.Builder
	for _, rss := range dateRange {
		concatenatedContent.WriteString(fmt.Sprintf("[标题]：%s\n", rss.Title))
		// 使用正则表达式删除连续的空格，保留单个空格
		content := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(rss.Content), " ")
		concatenatedContent.WriteString(fmt.Sprintf("[内容]：%s\n", content))
		concatenatedContent.WriteString(fmt.Sprintf("[链接]：%s\n\n", rss.Link))
	}
	fmt.Print(concatenatedContent.String())
}

func Test_rssContentDao_FindByID(t *testing.T) {
	llmPool := llm.NewLLMPool()
	ctx := context.Background()
	get, err := llmPool.Get(ctx)
	if err != nil {
		log.WithCtx(ctx).Errorf("get llm error: %w", err)
		return
	}
	engine, err := rag.NewEngine(ctx, get)
	if err != nil {
		t.Log("new engine error: %w", err)
		return
	}
	h, err := NewRssContentDao(models.GetDb()).FindAll(ctx)
	if err != nil {
		t.Log("find by 24h error: %w", err)
		return
	}
	for i, content := range h {
		err = engine.AddRssContent(ctx, content)
		if err != nil {
			t.Log("add rss content error: %w", err)
			return
		}
		log.WithCtx(ctx).Infof("add rss content: %d", i)
	}

	// 检索
	//var query string
	//for {
	//	_, err := fmt.Scan(&query)
	//	if err != nil {
	//		log.WithCtx(ctx).Fatal(err)
	//		continue
	//	}
	//
	//	rsp, err := engine.Query(ctx, query)
	//	if err != nil {
	//		log.WithCtx(ctx).Fatal(err)
	//	}
	//
	//	// 流式输出
	//	for {
	//		output, err := rsp.Recv()
	//		if err == io.EOF {
	//			break
	//		}
	//		if err != nil {
	//			log.WithCtx(ctx).Fatal(err)
	//		}
	//		fmt.Print(output.Content)
	//	}
	//}
}
