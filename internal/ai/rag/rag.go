package rag

import (
	"context"
	"fmt"
	"strings"

	"podcast/internal/ai/embedding"
	"podcast/internal/ai/pgvector"

	"podcast/internal/database/models"
	"podcast/pkg"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/youcd/toolkit/log"
	"gorm.io/gorm"
)

var systemPrompt = `
# Role: Student Learning Assistant

# Language: Chinese

- When providing assistance:
  • Be clear and concise
  • Include practical examples when relevant
  • Reference documentation when helpful
  • Suggest improvements or next steps if applicable

here's documents searched for you:
==== doc start ====
	  {documents}
==== doc end ====
`

// Engine rag 引擎
type Engine struct {
	Dimension  int64 // 嵌入维度
	db         *gorm.DB
	pgVector   *pgvector.PgVector
	embedder   *embedding.Embedder
	FileLoader *file.FileLoader     // 文件加载器
	Splitter   document.Transformer // 文本分割器
	LLM        *qwen.ChatModel      // 大模型
	cfg        *types.RagConfig
}

// NewEngine 创建 rag 引擎
func NewEngine(ctx context.Context, llmInfo *types.LLMInfo, cfg *types.RagConfig) (*Engine, error) {
	// 获取数据库连接
	db := models.GetDb()

	// 初始化 embedder
	embedder, err := embedding.NewEmbedder(ctx, cfg.Embedding)
	if err != nil {
		return nil, err
	}

	// 初始化 pgvector
	pgVector := pgvector.NewPgVector(ctx, cfg.PgVector)

	// 创建 rss_vectors 表（如果不存在）
	if err := pgVector.CreateCollection(ctx, cfg.PgVector.RssCollection); err != nil {
		pgVector.Close(ctx)
		return nil, fmt.Errorf("创建 rss_vectors 表失败: %w", err)
	}

	// 初始化 fileloader
	fileLoader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: true,
		Parser:      nil,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("NewEngine failed, init fileloader err: %v\n", err)
		return nil, err
	}

	// 初始化 splitter
	spliter, err := NewRecursiveSplitter(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化 llm
	chatModel, err := pkg.NewChatModel(ctx, llmInfo, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Dimension:  cfg.PgVector.Dimension,
		db:         db,
		pgVector:   pgVector,
		embedder:   embedder,
		FileLoader: fileLoader,
		Splitter:   spliter,
		LLM:        chatModel,
		cfg:        cfg,
	}, nil
}

// AddFile 添加文件
func (e *Engine) AddFile(ctx context.Context, filepath string) error {
	// 加载文件
	docs, err := e.FileLoader.Load(ctx, document.Source{
		URI: filepath,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("CreateFileIndex failed, load file err: %v\n", err)
		return err
	}
	log.WithCtx(ctx).Infof("docs: %#v", docs)
	// 分割文本
	docs, err = e.Splitter.Transform(ctx, docs)
	if err != nil {
		log.WithCtx(ctx).Errorf("CreateFileIndex failed, split text err: %v\n", err)
		return err
	}

	// 为每个文档生成唯一 id
	for _, d := range docs {
		uuid, _ := uuid.NewUUID()
		d.ID = uuid.String()
		log.WithCtx(ctx).Errorf("docs: %#v", d.String())
	}

	// 存储索引
	log.WithCtx(ctx).Infof("AddFile completed, docs: %d", len(docs))
	return nil
}

// Query 查询
func (e *Engine) Query(ctx context.Context, query string) (*schema.StreamReader[*schema.Message], error) {
	log.WithCtx(ctx).Infof("Query 查询:  %s ", strings.TrimSpace(query))

	// 生成查询向量
	queryVector, err := e.embedder.CreateEmbeddings(ctx, query)
	if err != nil {
		log.WithCtx(ctx).Errorf("Query failed, create embedding err: %v", err)
		return nil, err
	}

	// 使用 pgvector 进行语义搜索
	docs, err := e.searchByVector(ctx, queryVector, 5)
	if err != nil {
		log.WithCtx(ctx).Errorf("Query failed, search by vector err: %v", err)
		return nil, err
	}

	for _, doc := range docs {
		log.WithCtx(ctx).Infow("Retrieve", "metaData", doc.Content)
	}

	// 生成 prompt
	promptTempalte := prompt.FromMessages(schema.FString, []schema.MessagesTemplate{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage("question: {content}"),
	}...)
	message, err := promptTempalte.Format(ctx, map[string]any{
		"content":   query,
		"documents": docs,
	})
	if err != nil {
		log.WithCtx(ctx).Errorf("Query failed, format prompt err: %v\n", err)
		return nil, err
	}

	// 调用 llm
	return e.LLM.Stream(ctx, message)
}

// searchByVector 使用向量进行语义搜索
func (e *Engine) searchByVector(ctx context.Context, queryVector []float32, limit int) ([]*schema.Document, error) {
	collectionName := e.cfg.PgVector.RssCollection

	query := fmt.Sprintf(`
		SELECT id, title, md5, date, content, 1 - (embedding <=> $1::vector) as score
		FROM %s
		ORDER BY embedding <=> $1::vector
		LIMIT $2
	`, collectionName)

	var results []struct {
		ID      int     `gorm:"column:id"`
		Title   string  `gorm:"column:title"`
		MD5     string  `gorm:"column:md5"`
		Date    string  `gorm:"column:date"`
		Content string  `gorm:"column:content"`
		Score   float32 `gorm:"column:score"`
	}

	err := e.pgVector.GetDB().Raw(query, vectorToString(queryVector), limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	var docs []*schema.Document
	for _, r := range results {
		doc := &schema.Document{
			Content: r.Content,
			MetaData: map[string]any{
				"id":    r.ID,
				"title": r.Title,
				"md5":   r.MD5,
				"score": r.Score,
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// AddRssContent 添加Rss内容
func (e *Engine) AddRssContent(ctx context.Context, rss *models.RssContent) error {
	log.WithCtx(ctx).Infow("AddRssContent", "title", rss.Title)

	// 生成内容向量
	contentVector, err := e.embedder.CreateEmbeddings(ctx, rss.Content)
	if err != nil {
		log.WithCtx(ctx).Errorf("AddRssContent failed, create embedding err: %v", err)
		return err
	}

	// 存储到 PostgreSQL
	collectionName := e.cfg.PgVector.RssCollection
	query := fmt.Sprintf(`
		INSERT INTO %s (title, md5, date, content, embedding)
		VALUES ($1, $2, $3, $4, $5::vector)
		ON CONFLICT (md5) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			date = EXCLUDED.date
	`, collectionName)

	err = e.pgVector.GetDB().Exec(query, rss.Title, rss.MD5, rss.Date, rss.Content, vectorToString(contentVector)).Error
	if err != nil {
		return fmt.Errorf("插入数据失败: %w", err)
	}

	log.WithCtx(ctx).Infow("AddRssContent completed", "title", rss.Title)
	return nil
}

// vectorToString 将向量转换为 PostgreSQL 格式
func vectorToString(vector []float32) string {
	result := "["
	for i, v := range vector {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", v)
	}
	result += "]"
	return result
}

// Close 关闭引擎资源
func (e *Engine) Close(ctx context.Context) {
	if e.pgVector != nil {
		e.pgVector.Close(ctx)
	}
}

// rss_vectors 表结构（用于存储 RSS 内容和向量）
// CREATE TABLE IF NOT EXISTS rss_vectors (
//     id SERIAL PRIMARY KEY,
//     title VARCHAR(255) NOT NULL,
//     md5 VARCHAR(32) NOT NULL UNIQUE,
//     date TIMESTAMP NOT NULL,
//     content TEXT,
//     vector vector(1024)
// );

// 修改 CreateCollection 以支持 content 字段
func init() {
	// 这个函数会在包加载时执行，但我们需要修改 CreateCollection 来添加 content 字段
}
