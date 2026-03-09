package mcp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/youcd/toolkit/log"
)

type MCPServer struct {
	*server.MCPServer
	token string
}

func NewMCPServer(token string) *MCPServer {
	s := server.NewMCPServer(
		"YCD-MCP",
		"1.0.0",
		server.WithLogging(),
		server.WithInstructions("新闻咨询助手，请输入指令进行操作。"),
	)

	return &MCPServer{s, token}
}

func (s *MCPServer) RunWithGin(router *gin.Engine) {
	stream := server.NewStreamableHTTPServer(s.MCPServer, server.WithLogger(log.GetLogger()))
	s.Init()
	// 将MCP处理程序注册到Gin路由器
	router.Any("/mcp", func(c *gin.Context) {
		// 应用token验证
		token := c.Query("token")
		if token != s.token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 转发请求给MCP处理器
		stream.ServeHTTP(c.Writer, c.Request)
	})
}

func (s *MCPServer) Init() {
	s.RegisterTool(search24HRss, Search24HRss).
		RegisterTool(rssCategories, RssCategories).
		RegisterTool(getCurrentTime, GetCurrentTime)
	//s.RegisterTool(ragSearch, RagSearch)
	ctx := context.Background()
	proxy := InitMcpProxy(ctx)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	for name, c := range proxy.clientMap {
		to, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.WithCtx(ctx).Errorf("获取 %s 的工具列表失败: %v", name, err)
			continue
		}

		for _, t := range to.Tools {
			newTol := t
			newTol.Name = name + "_" + t.Name

			log.WithCtx(ctx).Debugf("注册工具: %s", newTol.Name)
			s.RegisterTool(
				newTol,
				handler(c, name),
			)
		}
	}
}
func handler(client *client.Client, name string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := request
		if strings.HasPrefix(request.Params.Name, name+"_") {
			params := request.Params
			params.Name = strings.TrimPrefix(params.Name, name+"_")
			req.Params = params
		}
		tool, err := client.CallTool(ctx, req)
		if err != nil {
			log.WithCtx(ctx).Errorf("调用工具失败: %v", err)
			return nil, err
		}
		return tool, nil
	}
}
func (s *MCPServer) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) *MCPServer {
	s.AddTool(tool, handler)
	return s
}
