package mcp

import (
	"context"
	"net/http"
	"podcast/pkg/types"
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
	token    string
	ragCfg   *types.RagConfig
	MCPProxy map[string]*types.Mcp
}

func NewMCPServer(token string, cfg *types.RagConfig, MCPProxy map[string]*types.Mcp) *MCPServer {
	s := server.NewMCPServer(
		"YCD-MCP",
		"1.0.0",
		server.WithLogging(),
		server.WithInstructions("新闻咨询助手，请输入指令进行操作。"),
	)

	return &MCPServer{s, token, cfg, MCPProxy}
}

func (m *MCPServer) RunWithGin(router *gin.Engine) {
	stream := server.NewStreamableHTTPServer(m.MCPServer, server.WithLogger(log.GetLogger()))
	m.Init(m.MCPProxy)
	// 将MCP处理程序注册到Gin路由器
	router.Any("/mcp", func(c *gin.Context) {
		// 应用token验证
		token := c.Query("token")
		if token != m.token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 转发请求给MCP处理器
		stream.ServeHTTP(c.Writer, c.Request)
	})
}

func (m *MCPServer) Init(mcpProxyConfig map[string]*types.Mcp) {
	m.RegisterTool(search24HRss, Search24HRss).
		RegisterTool(rssCategories, RssCategories).
		RegisterTool(getCurrentTime, GetCurrentTime)
	//m.RegisterTool(ragSearch, RagSearch)
	ctx := context.Background()
	proxy := InitMcpProxy(ctx, mcpProxyConfig)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	for name, c := range proxy.clientMap {
		to, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.WithCtx(ctx).Errorf("获取 %m 的工具列表失败: %v", name, err)
			continue
		}

		for _, t := range to.Tools {
			newTol := t
			newTol.Name = name + "_" + t.Name

			log.WithCtx(ctx).Debugf("注册工具: %m", newTol.Name)
			m.RegisterTool(
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
func (m *MCPServer) RegisterTool(tool mcp.Tool, handler server.ToolHandlerFunc) *MCPServer {
	m.AddTool(tool, handler)
	return m
}
