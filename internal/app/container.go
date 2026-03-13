package app

import (
	"context"
	"fmt"

	"podcast/config"
	"podcast/internal/ai/agent"
	"podcast/internal/ai/embedding"
	"podcast/internal/ai/llm"
	"podcast/internal/ai/milvus"
	"podcast/internal/database/dao"
	"podcast/internal/service"

	"github.com/youcd/toolkit/log"
	"gorm.io/gorm"
)

// Container 依赖注入容器：持有 config、db、DAO、Service，供路由与 cmd 使用
type Container struct {
	Cfg         *config.Config
	DB          *gorm.DB
	User        dao.UserDao
	Rss         dao.RssContentDao
	KeyInfo     dao.KeyInfoDao
	Report      dao.ReportDao
	Chat        *service.ChatService
	UserSvc     *service.UserService
	RssSvc      *service.RssService
	KeyInfoSvc  *service.KeyInfoService
	ReportSvc   *service.ReportService
	PromptSvc   *service.PromptService
	TemplateSvc *service.TemplateService
	// AI 组件
	Embedding *embedding.Embedder
	LLMPool   *llm.LLMPool
	Milvus    *milvus.Milvus

	// RAG Agent 配置
	MCPConfig *agent.MCPConfig
}

// New 从配置与 DB 构建容器（DAO、Service 均通过构造函数注入）
// 返回容器实例和错误，避免 panic
func New(ctx context.Context, cfg *config.Config, db *gorm.DB) (*Container, error) {
	c := &Container{Cfg: cfg, DB: db}

	c.User = dao.NewUserDao(db)
	c.Rss = dao.NewRssContentDao(db)
	c.KeyInfo = dao.NewKeyInfoDao(db)
	c.Report = dao.NewReportDao(db)
	sessionsDAO := dao.NewChatSessionsDAO(db)
	messagesDAO := dao.NewMessagesDAO(db)
	c.Chat = service.NewChatService(sessionsDAO, messagesDAO)

	token := ""
	if cfg.Global != nil {
		token = cfg.Global.Token
	}
	podcastDir := ""
	if cfg.Global != nil {
		podcastDir = cfg.Podcast.Dir
	}

	c.UserSvc = service.NewUserService(c.User, token)
	c.RssSvc = service.NewRssService(c.Rss)
	c.KeyInfoSvc = service.NewKeyInfoService(c.KeyInfo)
	c.ReportSvc = service.NewReportService(c.Report, podcastDir)
	c.PromptSvc = service.NewPromptService(c.KeyInfoSvc)
	c.TemplateSvc = service.NewTemplateService(c.KeyInfoSvc)

	// 初始化 AI 组件
	emb, err := embedding.NewEmbedder(ctx, cfg.Vector.Embedding)
	if err != nil {
		return nil, fmt.Errorf("初始化 Embedding 失败: %w", err)
	}
	c.Embedding = emb

	// 转换 LLM 配置
	c.LLMPool = llm.NewLLMPool(cfg.LLM)

	c.Milvus = milvus.NewMilvus(context.Background(), cfg.Database.Milvus)
	c.MCPConfig = &agent.MCPConfig{
		HostPort: cfg.Global.HostPort,
		Token:    cfg.Global.Token,
	}
	return c, nil
}

// Close 关闭容器中的资源，实现 io.Closer 接口
func (c *Container) Close(ctx context.Context) error {
	var errs []error

	// 关闭 Milvus 连接
	if c.Milvus != nil {
		c.Milvus.Close(ctx)
	}

	// 关闭数据库连接
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("获取数据库连接失败: %w", err))
		} else {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭数据库连接失败: %w", err))
			}
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			log.WithCtx(ctx).Error(e)
		}
		return fmt.Errorf("关闭容器时发生 %d 个错误", len(errs))
	}
	return nil
}
