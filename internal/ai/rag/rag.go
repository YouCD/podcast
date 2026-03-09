package rag

import (
	"context"
	"strings"

	"podcast/config"
	"podcast/internal/database/models"
	"podcast/pkg"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	milvus2Ret "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/youcd/toolkit/log"
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
	Dimension    int64 // 嵌入维度
	milvusClient *milvusclient.Client
	FileLoader   *file.FileLoader      // 文件加载器
	Splitter     document.Transformer  // 文本分割器
	Retriever    *milvus2Ret.Retriever // redis 检索器
	Indexer      *milvus2.Indexer      // redis 索引器
	LLM          *qwen.ChatModel       // 大模型
}

// NewEngine 创建 rag 引擎
func NewEngine(ctx context.Context, llmInfo *types.LLMInfo) (*Engine, error) {
	// 初始化 redis
	milvusClient, err := milvusclient.New(ctx, getMlvusClientConfig())
	if err != nil {
		return nil, err
	}

	// 初始化 embedder
	embedder, err := NewQwenEmbedder(ctx)
	if err != nil {
		return nil, err
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
	//spliter, err := NewMarkdownSplitter(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//spliter, err := NewSemanticSplitter(ctx, embedder)
	//if err != nil {
	//	return nil, err
	//}
	spliter, err := NewRecursiveSplitter(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化 retriever
	retriever, err := NewMilvusRetriever(ctx, embedder, 10)
	if err != nil {
		return nil, err
	}

	// 初始化 indexer
	indexer, err := NewMilvusIndexer(ctx, embedder)
	if err != nil {
		return nil, err
	}

	// 初始化 llm

	chatModel, err := pkg.NewChatModel(ctx, llmInfo, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Dimension:    config.Cfg.Database.Milvus.Dimension,
		milvusClient: milvusClient,
		FileLoader:   fileLoader,
		Splitter:     spliter,
		Retriever:    retriever,
		Indexer:      indexer,
		LLM:          chatModel,
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

	// 初始化向量索引
	if err := InitMilvusVectorIndex(ctx, e.Indexer, docs); err != nil {
		log.WithCtx(ctx).Errorf("CreateFileIndex failed, init vector index err: %v\n", err)
		return err
	}

	// 存储索引
	if _, err := e.Indexer.Store(ctx, docs); err != nil {
		log.WithCtx(ctx).Errorf("CreateFileIndex failed, store index err: %v\n", err)
		return err
	}
	return nil
}

// Query 查询
func (e *Engine) Query(ctx context.Context, query string) (*schema.StreamReader[*schema.Message], error) {
	log.WithCtx(ctx).Infof("Query 查询:  %s ", strings.TrimSpace(query))
	// 检索
	docs, err := e.Retriever.Retrieve(ctx, query)
	if err != nil {
		log.WithCtx(ctx).Errorf("Query failed, retrieve err: %v\n", err)
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

// AddRssContent 添加Rss内容
func (e *Engine) AddRssContent(ctx context.Context, rss *models.RssContent) error {
	log.WithCtx(ctx).Infow("AddRssContent", "rss", rss)
	doc := &schema.Document{
		Content: rss.Content,
		MetaData: map[string]any{
			"_source": rss.Link,
			"title":   rss.Title,
			"md5":     rss.MD5,
		},
	}

	docs := []*schema.Document{
		doc,
	}

	// 分割文本
	docs, err := e.Splitter.Transform(ctx, docs)
	if err != nil {
		log.WithCtx(ctx).Errorf("Splitter failed, split text err: %v", err)
		return err
	}
	// 为每个文档生成唯一 id
	for _, d := range docs {
		uuid, _ := uuid.NewUUID()
		d.ID = uuid.String()
		log.WithCtx(ctx).Infof("docs: %#v", d.String())
	}
	// 初始化向量索引
	if err := InitMilvusVectorIndex(ctx, e.Indexer, docs); err != nil {
		log.WithCtx(ctx).Errorf("CreateFileIndex failed, init vector index err: %v", err)
		return err
	}
	log.WithCtx(ctx).Infow("AddRssContent", "docs", len(docs))

	return nil
}
