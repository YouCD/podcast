package app

import (
	"podcast/config"
	"podcast/internal/database/dao"
	"podcast/internal/service"

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
}

// New 从配置与 DB 构建容器（DAO、Service 均通过构造函数注入）
func New(cfg *config.Config, db *gorm.DB) *Container {
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
		podcastDir = cfg.Global.PodcastDir
	}

	c.UserSvc = service.NewUserService(c.User, token)
	c.RssSvc = service.NewRssService(c.Rss)
	c.KeyInfoSvc = service.NewKeyInfoService(c.KeyInfo)
	c.ReportSvc = service.NewReportService(c.Report, podcastDir)
	c.PromptSvc = service.NewPromptService(c.KeyInfoSvc)
	c.TemplateSvc = service.NewTemplateService(c.KeyInfoSvc)

	return c
}
