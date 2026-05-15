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
	search24HRss = mcp.NewTool("news_search", mcp.WithDescription("功能：获取24H的新闻内容"),
		mcp.WithString("categories", mcp.Description("新闻的类别")),
	)
	rssCategories  = mcp.NewTool("news_categories", mcp.WithDescription("功能：获取新闻类别"))
	getCurrentTime = mcp.NewTool("get_current_time", mcp.WithDescription(`功能：获取当前时间, 格式为: "2025-11-22 15:04:05"`))
	// ragSearch      = mcp.NewTool("rag_search", mcp.WithDescription(`功能：从向量数据库中检索相关信息，支持语义搜索`),
	//	mcp.WithString("query", mcp.Description("用户的问题")))
)

func Search24HRss(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rssDao := dao.NewRssContentDao(models.GetDb())
	categories := request.GetString("categories", "科技")
	posts, err := rssDao.FindByCategory24H(ctx, categories)
	if err != nil {
		return nil, fmt.Errorf("FindByCategoryAndDate,err:%w", err)
	}

	var buffer strings.Builder
	for _, post := range posts {
		buffer.WriteString(fmt.Sprintf(`标题：%s
内容：%s
link：%s
`, post.Title, post.Content, post.Link))
	}
	log.WithCtx(ctx).Debugw("Search24HRss", "内容", buffer.String())
	return mcp.NewToolResultText(buffer.String()), nil
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
