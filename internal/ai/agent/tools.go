package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/config"
	"strings"
	"sync"

	"podcast/internal/ai/rag"
	"podcast/pkg/dgraph"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
)

// MilvusSearchTool 向量检索工具
type MilvusSearchTool struct{}

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
	// 初始化 embedder
	embedder, err := rag.NewQwenEmbedder(ctx)
	if err != nil {
		return "", err
	}

	retriever, err := rag.NewMilvusRetriever(ctx, embedder, args.TopK)
	if err != nil {
		return "", err
	}

	docs, err := retriever.Retrieve(ctx, args.Query)
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

var (
	once sync.Once
	ts   []tool.BaseTool
)

func initMcpTool(ctx context.Context) ([]tool.BaseTool, error) {
	var lastErr error

	once.Do(func() {
		tr, err := transport.NewStreamableHTTP("http://" + config.Cfg.Global.HostPort + "/mcp?token=" + config.Cfg.Global.Token)
		//tr, err := transport.NewStreamableHTTP("https://rss.youcd.online/mcp?token=youcd")
		if err != nil {
			log.WithCtx(context.Background()).Errorf("Failed to init SSE transport : %v", err)
			lastErr = err
		}

		cli := client.NewClient(tr)
		err = cli.Start(ctx)
		if err != nil {
			lastErr = fmt.Errorf("MCP客户端启动失败: %w", err)
		}
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "rag-agent",
			Version: "1.0.0",
		}
		_, err = cli.Initialize(ctx, initRequest)
		if err != nil {
			lastErr = fmt.Errorf("MCP初始化失败: %w", err)
		}

		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			lastErr = fmt.Errorf("获取MCP工具失败: %w", err)
		}
		d, err := dgraph.New()
		if err != nil {
			lastErr = fmt.Errorf("初始化DGraph客户端失败: %w", err)
		}

		// 初始化工具
		tools = append(tools,
			&DGraphQueryTool{
				d: d,
			},
			&MilvusSearchTool{},
		)
		ts = tools
	})

	return ts, lastErr
}
