package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/pgvector"
	"podcast/pkg/types"

	"podcast/pkg/dgraph"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/youcd/toolkit/log"
)

const (
	// defaultTopK 未指定 top_k 时的默认检索数量
	defaultTopK = 5
	// maxTopK top_k 上限，防止单次检索结果过大撑爆上下文
	maxTopK = 20
	// maxContentLen 单条文档返回给 LLM 的正文长度上限（字符）
	maxContentLen = 800
	// minRelevance 最低相关度阈值（cosine similarity），低于此分数的结果丢弃，避免噪声进上下文
	minRelevance = 0.4
)

// PgVectorSearchTool 向量检索工具
// 基于 pgvector 对 rss_vectors 表（RSS 文章正文向量）做语义检索，
// 并 join rss_content 补充链接/分类，供答案溯源引用
type PgVectorSearchTool struct {
	embedder   *embedding.Embedder
	pgVector   *pgvector.PgVector
	collection string
}

// NewPgVectorSearchTool 创建向量检索工具
func NewPgVectorSearchTool(ctx context.Context, ragConfig *types.RagConfig) (*PgVectorSearchTool, error) {
	embedder, err := embedding.NewEmbedder(ctx, ragConfig.Embedding)
	if err != nil {
		return nil, err
	}

	pgVector := pgvector.NewPgVector(ctx, ragConfig.PgVector)

	return &PgVectorSearchTool{
		embedder:   embedder,
		pgVector:   pgVector,
		collection: ragConfig.PgVector.RssCollection,
	}, nil
}

// Close 释放工具持有的数据库连接
func (t *PgVectorSearchTool) Close(ctx context.Context) {
	if t.pgVector != nil {
		t.pgVector.Close(ctx)
	}
}

func (t *PgVectorSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	p := schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"query": {
			Desc:     "检索查询语句，将自动转换为向量",
			Required: true,
			Type:     schema.String,
		},
		"top_k": {
			Desc:     "返回结果数量，默认 5，上限 20",
			Required: true,
			Type:     schema.Integer,
		},
		"start_date": {
			Desc: "可选，按文档日期过滤的开始日期，格式 YYYY-MM-DD",
			Type: schema.String,
		},
		"end_date": {
			Desc: "可选，按文档日期过滤的结束日期，格式 YYYY-MM-DD",
			Type: schema.String,
		},
		"categories": {
			Desc: "可选，按分类过滤，用逗号分隔的字符串（如 \"科技,金融\"），也可用字符串数组（如 [\"科技\",\"金融\"]）",
			Type: schema.String,
		},
		"reason": {
			Desc:     "调用此工具原因，说明为什么需要向量检索",
			Required: true,
			Type:     schema.String,
		},
	})

	return &schema.ToolInfo{
		Name:        "pgvector_search",
		Desc:        "基于语义相似度检索历史知识库、文档、资料；可选按日期范围/分类过滤",
		ParamsOneOf: p,
	}, nil
}

func (t *PgVectorSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query      string      `json:"query"`
		TopK       int         `json:"top_k"`
		StartDate  string      `json:"start_date"`
		EndDate    string      `json:"end_date"`
		Categories interface{} `json:"categories"` // 兼容字符串（逗号分隔）和字符串数组
		Reason     string      `json:"reason"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Query) == "" {
		return "", fmt.Errorf("query 不能为空")
	}
	if args.TopK <= 0 {
		args.TopK = defaultTopK
	}
	if args.TopK > maxTopK {
		args.TopK = maxTopK
	}

	log.WithCtx(ctx).Infow("ToolCall", "name", "pgvector_search", "args", args, "原因", args.Reason)

	// 2. 预解析过滤参数（在调用 embedding API 之前 fail fast）
	var startDT, endDT *time.Time
	if s := strings.TrimSpace(args.StartDate); s != "" {
		dt, err := parseDate(s)
		if err != nil {
			return "", fmt.Errorf("start_date 格式错误: %w", err)
		}
		startDT = &dt
	}
	if s := strings.TrimSpace(args.EndDate); s != "" {
		dt, err := parseDate(s)
		if err != nil {
			return "", fmt.Errorf("end_date 格式错误: %w", err)
		}
		endDT = &dt
	}
	var categoryFilters []string
	switch v := args.Categories.(type) {
	case string:
		for _, c := range strings.Split(v, ",") {
			if c = strings.TrimSpace(c); c != "" {
				categoryFilters = append(categoryFilters, c)
			}
		}
	case []interface{}:
		for _, item := range v {
			if c, ok := item.(string); ok {
				if c = strings.TrimSpace(c); c != "" {
					categoryFilters = append(categoryFilters, c)
				}
			}
		}
	}

	// 1. 生成查询向量
	queryVector, err := t.embedder.CreateEmbeddings(ctx, args.Query)
	if err != nil {
		return "", fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 3. 组装查询（相关度阈值 + 可选日期/分类过滤）
	params := []interface{}{pgvector.VectorToString(queryVector)}
	where := []string{}

	params = append(params, minRelevance)
	where = append(where, fmt.Sprintf("1 - (v.embedding <=> $1::halfvec) >= $%d", len(params)))

	if startDT != nil {
		params = append(params, *startDT)
		where = append(where, fmt.Sprintf("v.date >= $%d", len(params)))
	}
	if endDT != nil {
		params = append(params, endDT.Add(24*time.Hour-time.Second))
		where = append(where, fmt.Sprintf("v.date <= $%d", len(params)))
	}
	for _, c := range categoryFilters {
		params = append(params, c)
		where = append(where, fmt.Sprintf("c.categories = $%d", len(params)))
	}
	params = append(params, args.TopK)

	query := fmt.Sprintf(`
		SELECT v.md5,
		       max(v.title) AS title,
		       max(v.date) AS date,
		       max(1 - (v.embedding <=> $1::halfvec)) AS score,
		       (array_agg(v.content ORDER BY v.embedding <=> $1::halfvec))[1] AS content,
		       c.link, c.categories, c.source
		FROM %s v
		LEFT JOIN rss_content c ON c.md5 = v.md5 AND c.deleted_at IS NULL
		WHERE %s
		GROUP BY v.md5, c.link, c.categories, c.source
		ORDER BY score DESC
		LIMIT $%d
	`, t.collection, strings.Join(where, " AND "), len(params))

	var results []struct {
		MD5        string    `gorm:"column:md5"`
		Title      string    `gorm:"column:title"`
		Date       time.Time `gorm:"column:date"`
		Content    string    `gorm:"column:content"`
		Score      float64   `gorm:"column:score"`
		Link       string    `gorm:"column:link"`
		Categories string    `gorm:"column:categories"`
		Source     string    `gorm:"column:source"`
	}

	if err := t.pgVector.GetDB().Raw(query, params...).Scan(&results).Error; err != nil {
		log.WithCtx(ctx).Errorf("pgvector_search failed: %v\n", err)
		return "", fmt.Errorf("向量检索失败: %w", err)
	}

	if len(results) == 0 {
		return "知识库中未检索到相关文档。建议更换关键词/同义词重试，或改用其他工具。", nil
	}

	// 3. 格式化结果（带编号，便于最终答案引用）
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("在知识库中检索到 %d 条相关文档（按相关度排序，阈值 %.2f）:\n\n", len(results), minRelevance))
	for i, r := range results {
		content := strings.TrimSpace(r.Content)
		if runes := []rune(content); len(runes) > maxContentLen {
			content = string(runes[:maxContentLen]) + "..."
		}
		buf.WriteString(fmt.Sprintf("[%d] 标题: %s\n", i+1, r.Title))
		buf.WriteString(fmt.Sprintf("    日期: %s  相关度: %.2f\n", r.Date.Format("2006-01-02"), r.Score))
		if r.Categories != "" {
			buf.WriteString(fmt.Sprintf("    分类: %s\n", r.Categories))
		}
		if r.Link != "" {
			buf.WriteString(fmt.Sprintf("    链接: %s\n", r.Link))
		}
		buf.WriteString(fmt.Sprintf("    正文（相关片段）: %s\n\n", content))
	}
	return buf.String(), nil
}

// parseDate 解析 YYYY-MM-DD 日期（按应用本地时区，config.go 里 set time.Local）
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", "2006/01/02", "2006.01.02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("期望 YYYY-MM-DD，输入: %s", s)
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
		Desc:        "查询实体关系，输入格式：{\"entities\": [\"实体1\", \"实体2\"],\"reason\":\"调用原因\"}",
		ParamsOneOf: p,
	}, nil
}

func (t *DGraphQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Entities []string `json:"entities"`
		Reason   string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", err
	}

	log.WithCtx(ctx).Infow("ToolCall", "name", "dgraph_query", "args", args, "原因", args.Reason)
	if len(args.Entities) == 0 {
		return "", fmt.Errorf("请输入实体名称")
	}
	var nodes []string
	for _, n := range args.Entities {
		one, err := t.d.QueryOne(ctx, n)
		if err != nil {
			log.WithCtx(ctx).Warnw("dgraph", "Entity", args.Entities, "error", err)
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
	tools    []tool.BaseTool
	mcpCli   *client.Client
	dgraph   *dgraph.Dgraph
	pgVector *PgVectorSearchTool
}

// NewToolManager 创建工具管理器（推荐使用）
func NewToolManager(ctx context.Context, cfg *MCPConfig, ragConfig *types.RagConfig) (*ToolManager, error) {
	tm := &ToolManager{}

	// 初始化 MCP 客户端
	// 将 0.0.0.0 替换为 127.0.0.1，因为 0.0.0.0 是绑定地址而非路由地址
	hostPort := cfg.HostPort
	if strings.Contains(hostPort, "0.0.0.0") {
		hostPort = strings.Replace(hostPort, "0.0.0.0", "127.0.0.1", 1)
	}
	tr, err := transport.NewStreamableHTTP("http://" + hostPort + "/mcp?token=" + cfg.Token)
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

	// 初始化 PgVector 检索工具
	pgVectorTool, err := NewPgVectorSearchTool(ctx, ragConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 PgVector 检索工具失败：%w", err)
	}
	tm.pgVector = pgVectorTool

	// 组装所有工具
	tm.tools = append(
		mcpTools,
		&DGraphQueryTool{d: d},
		pgVectorTool,
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

	// 关闭 PgVector 连接
	if tm.pgVector != nil {
		tm.pgVector.Close(ctx)
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭工具管理器时发生 %d 个错误", len(errs))
	}
	return nil
}
