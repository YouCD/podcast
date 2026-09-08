package workflow

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"podcast/pkg/types"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/youcd/toolkit/log"
)

func fetchFeeds(ctx context.Context, resource []*types.RSSSource) (*graphState, error) {
	state := &graphState{}
	state.Categorization = make(map[*types.RSSItem]string)
	log.WithCtx(ctx).Info("[阶段1/7] 开始抓取RSS源")
	for _, rss := range resource {
		items, err := crawlAndParseRSS(ctx, rss)
		if err != nil {
			state.Errors = append(state.Errors, err)
			continue
		}
		state.RawItems = append(state.RawItems, items...)
	}
	log.WithCtx(ctx).Infof("[阶段1/7] 抓取完成，共 %d 条", len(state.RawItems))
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
			date = toLocalTime(*item.PublishedParsed, item.Published)
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

// tzSuffixRe 匹配 RSS 日期字符串末尾的时区标识（Z / +0800 / +08:00 / GMT / UTC 等）
var tzSuffixRe = regexp.MustCompile(`(?i)(Z|[+-]\d{2}:?\d{2}|GMT|UTC)$`)

// hasTimeZoneSuffix 判断 RSS 原始日期字符串是否携带时区偏移
func hasTimeZoneSuffix(s string) bool {
	return tzSuffixRe.MatchString(strings.TrimSpace(s))
}

// toLocalTime 将 gofeed 解析出的 RSS 发布时间规范为本地时区（time.Local，默认 Asia/Shanghai）。
// gofeed 会把「无时区」的日期默认按 UTC 解析（如 10:48:31 -> 10:48 UTC，本地实为 10:48），
// 导致存库后比真实本地时间少 8 小时。这里分两种情况处理：
//   - 原始字符串无时区偏移：RSS 中的墙钟时间即本地时间，直接重建为 time.Local。
//   - 原始字符串带时区偏移：gofeed 已正确计算绝对时刻，仅需转成 time.Local 表示。
func toLocalTime(parsed time.Time, raw string) time.Time {
	if hasTimeZoneSuffix(raw) {
		return parsed.In(time.Local)
	}
	y, mo, d := parsed.Date()
	h, mi, s := parsed.Clock()
	return time.Date(y, mo, d, h, mi, s, parsed.Nanosecond(), time.Local)
}
