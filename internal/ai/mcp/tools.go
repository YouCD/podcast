package mcp

import (
	"context"
	"fmt"
	"io"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/rag"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"strings"
	"time"

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
	//ragSearch      = mcp.NewTool("rag_search", mcp.WithDescription(`功能：从向量数据库中检索相关信息，支持语义搜索`),
	//	mcp.WithString("query", mcp.Description("用户的问题")))

	rssDao = dao.NewRssContentDao(models.GetDb())
)

func Search24HRss(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	categories := request.GetString("categories", "科技")
	posts, err := rssDao.FindByCategory24H(ctx, categories)
	if err != nil {
		return nil, fmt.Errorf("FindByCategoryAndDate,err:%w", err)
	}

	var buffer strings.Builder
	for _, post := range posts {
		buffer.WriteString(fmt.Sprintf(`标题：%s
内容：%s

`, post.Title, post.Content))
	}
	log.WithCtx(ctx).Debugw("Search24HRss", "内容", buffer.String())
	return mcp.NewToolResultText(buffer.String()), nil
}

func RssCategories(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	allCategories, err := rssDao.FindByTodayCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("FindByTodayCategory,err:%w", err)
	}
	return mcp.NewToolResultText(strings.Join(allCategories, "|")), nil
}

func GetCurrentTime(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(time.Now().Format("2006-01-02 15:04:05")), nil
}

func RagSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")

	// 创建RAG引擎
	llmPool := llm.NewLLMPool()
	llmInfo, err := llmPool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM info: %w", err)
	}
	defer func() {
		llmPool.Put(ctx, llmInfo)
	}()
	// 初始化RAG引擎
	ragEngine, err := rag.NewEngine(ctx, llmInfo)
	if err != nil {
		return nil, fmt.Errorf("初始化RAG引擎失败: %w", err)
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
