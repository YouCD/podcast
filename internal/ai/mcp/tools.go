package mcp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"podcast/internal/ai/rag"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
	"go.uber.org/zap/buffer"
)

var (
	searchRss = mcp.NewTool(
		"news_search", mcp.WithDescription("功能：获取新闻内容，按类别与可选日期范围；不传日期时返回最近 24H 未读内容"),
		mcp.WithString("categories", mcp.Description("新闻类别，逗号分隔支持多值；缺值时 24H 时 default 科技，日期范围时 全部类别")),
		mcp.WithString("start_date", mcp.Description("开始日期 YYYY-MM-DD；缺值（且 end_date 也缺）时走最近 24H")),
		mcp.WithString("end_date", mcp.Description("结束日期 YYYY-MM-DD；缺值时等于 start_date")),
	)
	rssCategories  = mcp.NewTool("news_categories", mcp.WithDescription("功能：获取新闻类别"))
	getCurrentTime = mcp.NewTool("get_current_time", mcp.WithDescription(`功能：获取当前时间, 格式为: "2025-11-22 15:04:05"`))
	// ragSearch      = mcp.NewTool("rag_search", mcp.WithDescription(`功能：从向量数据库中检索相关信息，支持语义搜索`),
	//	mcp.WithString("query", mcp.Description("用户的问题")))
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
	if endOfDay.Sub(startOfDay) > 31*24*time.Hour {
		return nil, fmt.Errorf("日期范围超过 31 天，请缩小范围")
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
	return mcp.NewToolResultText(formatPosts(posts, true)), nil
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

func (m *MCPServer) RagSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")

	pool := m.llmPool
	llmInfo, err := pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 LLM 信息失败：%w", err)
	}
	defer pool.Put(ctx, llmInfo)
	// 初始化 RAG 引擎
	ragEngine, err := rag.NewEngine(ctx, llmInfo, m.ragCfg)
	if err != nil {
		return nil, fmt.Errorf("初始化 RAG 引擎失败：%w", err)
	}
	defer ragEngine.Close(ctx)
	stream, err := ragEngine.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("RAG查询失败: %w", err)
	}

	var buf buffer.Buffer
	for {
		select {
		case <-ctx.Done():
			// 客户端断开连接
			log.WithCtx(ctx).Debug("Client disconnected")
			return nil, nil
		default:
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return mcp.NewToolResultText(buf.String()), nil
				}
				return nil, fmt.Errorf("Error reading stream: %w", err)
			}

			if msg != nil && msg.Content != "" {
				if _, err := buf.WriteString(msg.Content); err != nil {
					return nil, fmt.Errorf("Error writing to buffer: %w", err)
				}
			}

			// 每次循环都检查连接状态
			if ctx.Err() != nil {
				return nil, nil
			}
		}
	}
}
