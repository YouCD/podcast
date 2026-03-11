package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/internal/ai/embedding"
	"podcast/pkg/types"
	"strings"

	"podcast/internal/ai/rag"
	"podcast/pkg/dgraph"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	milvus2Ret "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
)

// MilvusSearchTool 向量检索工具
type MilvusSearchTool struct {
	embedder  *dashscope.Embedder
	retriever *milvus2Ret.Retriever
}

// NewMilvusSearchTool 创建向量检索工具
func NewMilvusSearchTool(ctx context.Context, embedderCfg *types.Embedding, cfg *types.Milvus) (*MilvusSearchTool, error) {
	embedder, err := embedding.NewEmbedder(ctx, embedderCfg)
	if err != nil {
		return nil, err
	}

	retriever, err := rag.NewMilvusRetriever(ctx, embedder.Embedder, 10, cfg)
	if err != nil {
		return nil, err
	}

	return &MilvusSearchTool{
		embedder:  embedder.Embedder,
		retriever: retriever,
	}, nil
}

func (t *MilvusSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	p := schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"query": {
			Desc:     "检索查询语句，将自动转换为向量",
			Required: true,
			Type:     schema.String,
		},
		"top_k": {
			Desc:     "返回结果数量",
			Required: true,
			Type:     schema.Integer,
		},
		"reason": {
			Desc:     "调用此工具原因，说明为什么需要向量检索",
			Required: true,
			Type:     schema.String,
		},
	})

	return &schema.ToolInfo{
		Name:        "milvus_search",
		Desc:        "基于语义相似度检索历史知识库、文档、资料",
		ParamsOneOf: p,
	}, nil
}

func (t *MilvusSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query      string `json:"query"`
		Collection string `json:"collection"`
		TopK       int    `json:"top_k"`
		Reason     string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", err
	}

	log.WithCtx(ctx).Infow("ToolCall", "name", "milvus_search", "args", args, "原因", args.Reason)

	docs, err := t.retriever.Retrieve(ctx, args.Query)
	if err != nil {
		log.WithCtx(ctx).Errorf("Query failed, retrieve err: %v\n", err)
		return "", err
	}

	// 格式化结果
	var buf strings.Builder
	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf(`Content: %s
link: %s

`, doc.String(), doc.MetaData["_source"]))
		log.WithCtx(ctx).Debugw("ToolCall", "Content", doc.String())
	}

	return buf.String(), nil
}

// DGraphQueryTool 关系查询工具
type DGraphQueryTool struct {
	d *dgraph.Dgraph
}

func (t *DGraphQueryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	p := schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"entity": {
			Desc:     "实体名称",
			Required: true,
			Type:     schema.Array,
		},
		"reason": {
			Desc:     "调用此工具的原因，说明为什么需要关系查询",
			Required: true,
			Type:     schema.String,
		},
	})

	return &schema.ToolInfo{
		Name:        "dgraph_query",
		Desc:        "查询实体关系、知识图谱、关联路径。适用于查询'谁和谁的关系'、'A对B的影响'、'关联网络'等关系型问题",
		ParamsOneOf: p,
	}, nil
}

func (t *DGraphQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Entity []string `json:"entity"`
		Reason string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", err
	}

	log.WithCtx(ctx).Infow("ToolCall", "name", "dgraph_query", "args", args, "原因", args.Reason)
	if len(args.Entity) == 0 {
		return "", fmt.Errorf("请输入实体名称")
	}
	var nodes []string
	for _, n := range args.Entity {
		one, err := t.d.QueryOne(ctx, n)
		if err != nil {
			log.WithCtx(ctx).Warnw("dgraph", "Entity", args.Entity, "error", err)
			continue
		}
		nodes = append(nodes, one.Uid)
	}

	dql := fmt.Sprintf(`{
  q(func: uid(%s)) {
    uid
    dgraph.type
    expand(_all_) {
      uid
      name
      dgraph.type
      expand(_all_) {
        uid
        name
        dgraph.type
      }
    }
  }
}
`, strings.Join(nodes, ","))
	result, err := t.d.Predicates(ctx, dql)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`实例关系如下：
%s`, result), nil
}

// MCPConfig MCP 客户端配置
type MCPConfig struct {
	HostPort string
	Token    string
}

// ToolManager 工具管理器，管理所有 Agent 工具的生命周期
type ToolManager struct {
	tools  []tool.BaseTool
	mcpCli *client.Client
	dgraph *dgraph.Dgraph
	milvus *MilvusSearchTool
}

// NewToolManager 创建工具管理器（推荐使用）
func NewToolManager(ctx context.Context, cfg *MCPConfig, ragConfig *types.RagConfig) (*ToolManager, error) {
	tm := &ToolManager{}

	// 初始化 MCP 客户端
	tr, err := transport.NewStreamableHTTP("http://" + cfg.HostPort + "/mcp?token=" + cfg.Token)
	if err != nil {
		log.WithCtx(ctx).Errorf("Failed to init SSE transport : %v", err)
		return nil, fmt.Errorf("初始化 SSE 传输失败: %w", err)
	}

	cli := client.NewClient(tr)
	tm.mcpCli = cli

	err = cli.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("MCP客户端启动失败: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "rag-agent",
		Version: "1.0.0",
	}
	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("MCP初始化失败: %w", err)
	}

	// 获取 MCP 工具
	mcpTools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
	if err != nil {
		return nil, fmt.Errorf("获取MCP工具失败: %w", err)
	}

	// 初始化 DGraph
	d, err := dgraph.New(ragConfig.DgraphHost)
	if err != nil {
		return nil, fmt.Errorf("初始化DGraph客户端失败: %w", err)
	}
	tm.dgraph = d

	// 初始化 Milvus 检索工具
	milvusTool, err := NewMilvusSearchTool(ctx, ragConfig.Embedding, ragConfig.Milvus)
	if err != nil {
		return nil, fmt.Errorf("初始化 Milvus 检索工具失败：%w", err)
	}
	tm.milvus = milvusTool

	// 组装所有工具
	tm.tools = append(mcpTools,
		&DGraphQueryTool{d: d},
		milvusTool,
	)

	return tm, nil
}

// GetTools 获取所有工具
func (tm *ToolManager) GetTools() []tool.BaseTool {
	return tm.tools
}

// Close 关闭工具管理器，释放资源
func (tm *ToolManager) Close(ctx context.Context) error {
	var errs []error

	// 关闭 MCP 客户端
	if tm.mcpCli != nil {
		if err := tm.mcpCli.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 MCP 客户端失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭工具管理器时发生 %d 个错误", len(errs))
	}
	return nil
}

// initMcpTool 初始化 MCP 工具（兼容旧代码，推荐使用 NewToolManager）
// Deprecated: 请使用 NewToolManager
func initMcpTool(ctx context.Context, cfg *MCPConfig, ragConfig *types.RagConfig) ([]tool.BaseTool, error) {
	tm, err := NewToolManager(ctx, cfg, ragConfig)
	if err != nil {
		return nil, err
	}
	return tm.GetTools(), nil
}
