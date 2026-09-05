package mcp

import (
	"context"
	"errors"
	"sync"
	"time"

	"podcast/pkg/types"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
)

var (
	mcpProxy *McpProxy
	once     sync.Once
)

type McpProxy struct {
	clientMap map[string]*client.Client
	cacheTool map[string]string
	urlMap    map[string]*types.Mcp // 保存URL用于重连
	ctx       context.Context       // 保存上下文用于创建新客户端
	mutex     sync.RWMutex          // 保护clientMap的并发访问
}

func InitMcpProxy(ctx context.Context, mcpProxyConfig map[string]*types.Mcp) *McpProxy {
	clientMap := make(map[string]*client.Client)
	cacheTool := make(map[string]string)
	urlMap := make(map[string]*types.Mcp)

	once.Do(func() {
		for key, mcp := range mcpProxyConfig {
			log.WithCtx(ctx).Infof("正在初始化MCP客户端: %s", key)
			c, err := newClient(ctx, mcp)
			if err != nil {
				log.WithCtx(ctx).Errorf("Failed to init MCP client: %v", err)
				continue
			}
			clientMap[key] = c
		}
		mcpProxy = &McpProxy{
			clientMap: clientMap,
			cacheTool: cacheTool,
			urlMap:    urlMap,
			ctx:       ctx,
		}
	})
	log.WithCtx(ctx).Infof("启动MCP代理,代理数量: %d", len(clientMap))

	// 启动健康检查协程
	go mcpProxy.healthCheck()

	return mcpProxy
}

// healthCheck 定期检查客户端连接状态并尝试重连
func (m *McpProxy) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			log.WithCtx(context.Background()).Info("健康检查已停止")
			return
		case <-ticker.C:
			m.checkAndReconnect()
		}
	}
}

// checkAndReconnect 检查并重新连接失效的客户端
func (m *McpProxy) checkAndReconnect() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, url := range m.urlMap {
		// 尝试通过一个轻量级操作检查连接是否有效
		cc, exists := m.clientMap[name]
		if !exists {
			log.WithCtx(context.Background()).Warnf("客户端 %s 不存在，尝试创建新连接", name)
			c, err := newClient(m.ctx, url)
			if err != nil {
				log.WithCtx(context.Background()).Warnf("创建客户端 %s 失败: %v", name, err)
				continue
			}
			m.clientMap[name] = c
			log.WithCtx(context.Background()).Infof("客户端 %s 创建成功", name)
			continue
		}

		// 检查现有连接
		_, err := cc.ListTools(m.ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.WithCtx(context.Background()).Warnf("检测到客户端 %s 连接失效: %v，正在尝试重连...", name, err)

			// 尝试重新连接
			c, err := newClient(m.ctx, url)
			if err != nil {
				log.WithCtx(context.Background()).Warnf("重连客户端 %s 失败: %v", name, err)
				continue
			}
			m.clientMap[name] = c
			log.WithCtx(context.Background()).Infof("客户端 %s 重连成功", name)
		}
	}
}

func (m *McpProxy) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var tools []*mcp.Tool
	for clientName, mClient := range m.clientMap {
		log.WithCtx(context.Background()).Infof("正在获取 %s 的工具列表", clientName)
		listTools, err := mClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.WithCtx(context.Background()).Debugf("Failed to list tools for %s: %v", clientName, err)
			continue
		}
		for _, t := range listTools.Tools {
			m.cacheTool[t.Name] = clientName
			tools = append(tools, &t)
		}
	}
	return tools, nil
}

func (m *McpProxy) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.mutex.RLock()
	cName, ok := m.cacheTool[request.Params.Name]
	m.mutex.RUnlock()

	if !ok {
		return nil, errors.New("tool not found")
	}

	// 先尝试获取客户端
	mcpClient, err := m.GetClientByNme(cName)
	if err != nil {
		// 如果获取失败，尝试重新连接
		m.mutex.Lock()
		url, exists := m.urlMap[cName]
		if exists {
			log.WithCtx(context.Background()).Infof("客户端 %s 不存在，尝试重新连接", cName)
			c, err := newClient(ctx, url)
			if err != nil {
				log.WithCtx(context.Background()).Errorf("Failed to reconnect to %s: %v", cName, err)
				return nil, err
			}
			m.clientMap[cName] = c
			mcpClient = c
		}
		m.mutex.Unlock()
	}

	result, err := mcpClient.CallTool(ctx, request)
	if err != nil {
		// 如果调用失败，可能是连接问题，尝试重新连接
		m.mutex.Lock()
		if url, exists := m.urlMap[cName]; exists {
			log.WithCtx(context.Background()).Infof("调用工具失败，尝试重新连接 %s: %v", cName, err)
			c, err := newClient(ctx, url)
			if err != nil {
				log.WithCtx(context.Background()).Errorf("Failed to reconnect to %s: %v", cName, err)
				m.mutex.Unlock()
				return nil, err
			}
			m.clientMap[cName] = c
			m.mutex.Unlock()
			// 重试一次
			return c.CallTool(ctx, request)
		}
		m.mutex.Unlock()
	}
	return result, err
}

func (m *McpProxy) GetClientByNme(name string) (*client.Client, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if c, ok := m.clientMap[name]; ok {
		return c, nil
	}
	return nil, errors.New("client not found")
}

func newClient(ctx context.Context, mcpInfo *types.Mcp) (*client.Client, error) {
	tr, err := transport.NewStreamableHTTP(
		mcpInfo.URL,
		transport.WithHTTPHeaders(mcpInfo.Headers),
	)
	if err != nil {
		log.WithCtx(context.Background()).Errorf("Failed to init SSE transport for %s: %v", mcpInfo.URL, err)
		return nil, err
	}

	mcpClient := client.NewClient(tr)

	if err := mcpClient.Start(ctx); err != nil {
		log.WithCtx(context.Background()).Errorf("Failed to start transport for %s: %v", mcpInfo.URL, err)
		return nil, err
	}
	if _, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{}); err != nil {
		log.WithCtx(context.Background()).Errorf("MCP initialize error for %s: %v", mcpInfo.URL, err)
		return nil, err
	}
	return mcpClient, nil
}
