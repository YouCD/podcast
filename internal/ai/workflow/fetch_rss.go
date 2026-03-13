package workflow

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"podcast/pkg/types"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/youcd/toolkit/log"
)

func fetchFeeds(ctx context.Context, resource []*types.RSSSource) (*graphState, error) {
	state := &graphState{}
	state.Categorization = make(map[*types.RSSItem]string)
	var wg sync.WaitGroup
	log.WithCtx(ctx).Info("开始抓取RSS源")
	for _, rss := range resource {
		wg.Add(1)
		go func(rss *types.RSSSource) {
			defer wg.Done()
			items, err := crawlAndParseRSS(ctx, rss)
			if err != nil {
				state.Errors = append(state.Errors, err)
				return
			}
			state.RawItems = append(state.RawItems, items...)
		}(rss)
	}
	wg.Wait()
	return state, nil
}

// crawlAndParseRSS 爬取并解析RSS链接
func crawlAndParseRSS(ctx context.Context, rss *types.RSSSource) ([]*types.RSSItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(rss.URL, ctx)
	if err != nil {
		return nil, fmt.Errorf("解析RSS链接失败: %w", err)
	}

	items := make([]*types.RSSItem, 0)
	for _, item := range feed.Items {
		// 获取内容，优先使用Content字段，如果没有则使用Description字段
		content := CleanHTML(item.Content)
		if content == "" {
			content = CleanHTML(item.Description)
		}
		if content == "" {
			log.WithCtx(ctx).Warnw("content为空", "Rss", rss.Name, "Title", item.Title, "rss.URL", rss.URL, "link", item.Link)
			continue
		}
		if len([]rune(content)) < 20 {
			log.WithCtx(ctx).Debugw("content过短", "content", content, "Rss", rss.Name, "Title", item.Title, "rss.URL", rss.URL, "link", item.Link)
			continue
		}
		if item.Title == "" {
			log.WithCtx(ctx).Warnw("标题为空", "Rss", rss.Name, "Title", item.Title, "rss.URL", rss.URL, "原始Content", item.Content, "link", item.Link)
			continue
		}

		// 获取发布日期
		date := time.Now()
		if item.PublishedParsed != nil {
			date = *item.PublishedParsed
		}

		// 计算标题的MD5值用于去重
		//nolint:gosec
		md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(item.Title)))

		rssItem := types.RSSItem{
			Title:   item.Title,
			Content: content,
			Date:    date,
			Source:  rss.Name,
			Link:    item.Link,
			MD5:     md5Hash,
		}

		items = append(items, &rssItem)
	}

	return items, nil
}

func CleanHTML(input string) string {
	p := bluemonday.StrictPolicy()
	return p.Sanitize(input)
}
