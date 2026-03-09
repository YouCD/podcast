package mcp

import (
	"context"
	"fmt"
	"podcast/config"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/youcd/toolkit/log"
)

func init() {
	c, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.local.yaml")
	if err != nil {
		panic(err)
	}
	log.WithCtx(context.Background()).Infof("%#v", c)
}
func TestMcpProxy_Reconnect(t *testing.T) {
	// 创建一个带取消功能的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化代理
	proxy := InitMcpProxy(ctx)

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 测试获取工具列表
	tools, err := proxy.ListTools(ctx)

	for _, tool := range tools {
		fmt.Println(tool.Name, tool.Description)
		fmt.Println(tool.InputSchema)
	}

	assert.NoError(t, err)
	t.Logf("获取到 %d 个工具", len(tools))

	// 测试调用工具（如果有的话）
	if len(tools) > 0 {
		// 创建一个模拟的工具调用请求
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: tools[0].Name,
			},
		}

		// 注意：由于我们没有实际的工具实现，这个调用可能会失败
		// 但我们主要测试的是连接机制
		_, err := proxy.CallTool(ctx, request)
		// 错误是预期的，因为我们没有实际的工具实现
		t.Logf("调用工具结果: %v", err)
	}

	// 测试健康检查机制
	// 等待一段时间让健康检查运行
	time.Sleep(35 * time.Second)

	// 再次尝试获取工具列表
	toolsAfter, err := proxy.ListTools(ctx)
	assert.NoError(t, err)
	t.Logf("健康检查后获取到 %d 个工具", len(toolsAfter))
}
