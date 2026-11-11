package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
)

var (
	searchRss = mcp.NewTool(
		"news_search", mcp.WithDescription("功能：获取新闻内容，按类别与可选日期范围；不传日期时返回最近 24H 未读内容"),
		mcp.WithString("categories", mcp.Description("新闻类别，逗号分隔支持多值；缺值时 24H 时 default 科技，日期范围时 全部类别")),
		mcp.WithString("start_date", mcp.Description("开始日期 YYYY-MM-DD；缺值（且 end_date 也缺）时走最近 24H；日期范围不能超过 31 天")),
		mcp.WithString("end_date", mcp.Description("结束日期 YYYY-MM-DD；缺值时等于 start_date；与 start_date 间隔不能超过 31 天")),
	)
	rssCategories  = mcp.NewTool("news_categories", mcp.WithDescription("功能：获取新闻类别"))
	getCurrentTime = mcp.NewTool("get_current_time", mcp.WithDescription(`功能：获取当前时间, 格式为: "2025-11-22 15:04:05"`))
)

// ParseDate 解析 YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS 输入，按应用本地时区（config.go里 set time.Local）
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", "2006/01/02", "2006.01.02"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("日期格式错误，期望 YYYY-MM-DD，输入: %s", s)
}

// SearchRss 按类别与可选日期范围获取新闻内容
// 不传日期时走最近 24H（DAO里 created_at 近 24H 且 time_stay=0 未读语义，与旧 news_search 一致）
// 传日期时走 date 列整-day 闭区间窗口
func SearchRss(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rssDao := dao.NewRssContentDao(models.GetDb())

	categories := strings.TrimSpace(request.GetString("categories", ""))
	startRaw := strings.TrimSpace(request.GetString("start_date", ""))
	endRaw := strings.TrimSpace(request.GetString("end_date", ""))

	var posts []*models.RssContent

	// 24H 路径
	if startRaw == "" && endRaw == "" {
		targets := []string{"科技"}
		if categories != "" {
			targets = splitCategories(categories)
		}
		for _, c := range targets {
			one, err := rssDao.FindByCategory24H(ctx, c)
			if err != nil {
				return nil, fmt.Errorf("FindByCategory24H,err:%w", err)
			}
			posts = append(posts, one...)
		}
		log.WithCtx(ctx).Debugw("SearchRss", "mode", "24h", "categories", targets, "count", len(posts))
		return mcp.NewToolResultText(formatPosts(posts, false)), nil
	}

	// 日期范围路径
	var err error
	start, end := time.Now(), time.Now()
	if startRaw != "" {
		start, err = ParseDate(startRaw)
		if err != nil {
			return nil, err
		}
	}
	if endRaw != "" {
		end, err = ParseDate(endRaw)
		if err != nil {
			return nil, err
		}
	}

	// 整-day 窗口：[start 00:00, end 23:59:59]，DAO里 date <= endDate 闭区间
	startOfDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endOfDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location()).Add(24*time.Hour - time.Second)
	if endOfDay.Before(startOfDay) {
		return nil, fmt.Errorf("start_date must not be later than end_date")
	}
	// 模型规划宽范围是常见误规划，硬失败会触发无谓 replan；
	// 自动收敛到以 end 为锚点的最近 31 天，并在结果头部注明
	clamped := false
	if endOfDay.Sub(startOfDay) > 31*24*time.Hour {
		startOfDay = endOfDay.Add(-31*24*time.Hour + time.Second)
		clamped = true
		log.WithCtx(ctx).Infow("SearchRss 日期范围超过 31 天，自动收敛", "start", startOfDay, "end", endOfDay)
	}

	targets := splitCategories(categories)
	if len(targets) == 0 {
		posts, err = rssDao.FindByDateRange(ctx, startOfDay, endOfDay)
		if err != nil {
			return nil, fmt.Errorf("FindByDateRange,err:%w", err)
		}
	} else {
		// FindByDateRangeCategories 只 single 类别，comma-separated 时 loop merge
		for _, c := range targets {
			one, err := rssDao.FindByDateRangeCategories(ctx, startOfDay, endOfDay, c)
			if err != nil {
				return nil, fmt.Errorf("FindByDateRangeCategories,err:%w", err)
			}
			posts = append(posts, one...)
		}
	}

	log.WithCtx(ctx).Debugw("SearchRss", "mode", "range", "start", startOfDay, "end", endOfDay,
		"categories", targets, "count", len(posts))
	result := formatPosts(posts, true)
	if clamped {
		result = fmt.Sprintf("注意：请求的日期范围超过 31 天，已自动收敛为最近 31 天（%s ~ %s）\n", startOfDay.Format("2006-01-02"), endOfDay.Format("2006-01-02")) + result
	}
	return mcp.NewToolResultText(result), nil
}

// splitCategories 逗号分隔类别串转 slice，filter 空白项
func splitCategories(s string) []string {
	var out []string
	for _, c := range strings.Split(s, ",") {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// formatPosts 拼接工具输出；withDate 时附带日期与类别行，供模型引用时效
func formatPosts(posts []*models.RssContent, withDate bool) string {
	var b strings.Builder
	for _, post := range posts {
		if withDate {
			b.WriteString(fmt.Sprintf(`标题：%s
日期：%s
类别：%s
内容：%s
link：%s
`, post.Title, post.Date.Format("2006-01-02 15:04:05"), post.Categories, post.Content, post.Link))
			continue
		}
		b.WriteString(fmt.Sprintf(`标题：%s
内容：%s
link：%s
`, post.Title, post.Content, post.Link))
	}
	return b.String()
}

func RssCategories(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rssDao := dao.NewRssContentDao(models.GetDb())
	allCategories, err := rssDao.FindByTodayCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("FindByTodayCategory,err:%w", err)
	}
	return mcp.NewToolResultText(strings.Join(allCategories, "|")), nil
}

func GetCurrentTime(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(time.Now().Format("2006-01-02 15:04:05")), nil
}
