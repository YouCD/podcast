package routes

import (
	"embed"
	"net/http"
	"strings"

	"podcast/internal/ai/agent"
	"podcast/pkg/types"

	"podcast/internal/ai/mcp"
	"podcast/internal/app"
	"podcast/web/dist"
	"podcast/web/handlers"

	"github.com/gin-gonic/gin"
	"github.com/youcd/toolkit/pprof"
)

func StaticServe(fs embed.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(fs))
	return func(ctx *gin.Context) {
		if strings.Contains(ctx.Request.URL.Path, "/assets") || ctx.Request.URL.Path == "/" {
			fileServer.ServeHTTP(ctx.Writer, ctx.Request)
			ctx.Abort()
			return
		}
	}
}

// SetupRouter 使用依赖注入容器创建路由（cfg、db、DAO、Service 均由 container 提供）
func SetupRouter(container *app.Container) *gin.Engine {
	r := gin.Default()

	rssHandler := handlers.NewRssHandler(container.RssSvc, container.Cfg.Global.ContentLen, container.Cfg.Database.Dgraph)
	userHandler := handlers.NewUserHandler(container.UserSvc)
	reportsHandler := handlers.NewReportsHandler(container.ReportSvc, container.Cfg.Podcast, container.LLMPool)
	keyinfosHandler := handlers.NewKeyInfoHandler(container.KeyInfoSvc)
	ragCfg := &types.RagConfig{
		PgVector:   container.Cfg.Database.PostgreSQL.ToPgVector(),
		Embedding:  container.Cfg.Vector.Embedding,
		DgraphHost: container.Cfg.Database.Dgraph,
	}
	chatHandler := handlers.NewChatHandler(container.Chat, &agent.RAGAgentConfig{
		MCP:       container.MCPConfig,
		RagConfig: ragCfg,
		LLMPool:   container.LLMPool,
	}, container.LLMPool)
	promptHandler := handlers.NewPromptHandler(container.PromptSvc)
	templateHandler := handlers.NewTemplateHandler(container.TemplateSvc)

	token := ""
	logLevel := "info"
	if container.Cfg != nil && container.Cfg.Global != nil {
		token = container.Cfg.Global.Token
		logLevel = container.Cfg.Global.LogLevel
	}
	mcp.NewMCPServer(token, ragCfg, container.Cfg.MCPProxy, container.LLMPool).RunWithGin(r)
	r.Use(StaticServe(dist.Dist))
	r.Use(handlers.RequestIDMiddleware())
	r.Use(handlers.CorsMiddleware())
	r.GET("/rss", rssHandler.ExportRss24H)

	if strings.ToLower(logLevel) == "debug" {
		pprof.RegisterHandlers(func(pattern string, handler func(http.ResponseWriter, *http.Request)) {
			r.GET(pattern, gin.WrapF(handler))
		})
	}

	// 公开访问的路由 - 不需要认证
	r.GET("/api/feed", rssHandler.GetRssContent)
	r.GET("/api/feed/:id", rssHandler.GetRssContentByID)
	r.GET("/api/feed/:id/llm_html", rssHandler.GetLLMHTMLByID)
	r.GET("/api/feed/categories", rssHandler.GetAllCategories)
	r.GET("/api/feed/categories/:category/24h", rssHandler.GetRssByCategory24H)
	r.GET("/api/feed/read24h", rssHandler.GetRss24H)
	r.GET("/api/feed/status", rssHandler.Status)
	r.GET("/api/feed/not_read", rssHandler.NotRead)

	// 用户登录接口 - 不需要认证
	user := r.Group("/api/user")
	{
		user.POST("/login", userHandler.Login)
	}

	// Report相关路由 - 需要认证
	reports := r.Group("/api/reports")
	{
		// 获取所有report列表，但不包含LLMResult
		reports.GET("/", reportsHandler.GetReports)
		// 根据ID获取指定report的LLMResult
		reports.GET("/:id/llm_result", reportsHandler.GetLLMResultByID)
		// 根据ID获取指定report详情
		reports.GET("/:id/detail", reportsHandler.GetReportDetailByID)
		reports.GET("/:id/play", reportsHandler.PlayByID)
		// 手动生成日报
		reports.GET("/:id/daily_report", reportsHandler.GenDailyReport)
	}

	// 需要认证的路由组
	protected := r.Group("/")
	protected.Use(handlers.JwtMiddleware(container.UserSvc))
	{
		// rss相关路由 - 需要认证
		feed := protected.Group("/api/feed")
		{
			feed.PUT("/:id", rssHandler.UpdateRssContent)
			feed.POST("/time_stay", rssHandler.HandleTimeStayAndMD5)
		}

		// KeyInfo相关路由 - 需要认证
		keyinfos := protected.Group("/api/keyinfos")
		{
			keyinfos.POST("/", keyinfosHandler.CreateKeyInfo)
			keyinfos.GET("/", keyinfosHandler.GetAllKeyInfos)
			keyinfos.GET("/:id", keyinfosHandler.GetKeyInfoByID)
			keyinfos.GET("/key_name/:key_name", keyinfosHandler.GetKeyInfoByKeyName)
			keyinfos.PUT("/:id", keyinfosHandler.UpdateKeyInfo)
			keyinfos.DELETE("/:id", keyinfosHandler.DeleteKeyInfo)
			// keyinfos.GET("/genre/:genre", keyinfosHandler.GetKeyInfosByGenre)
			// keyinfos.GET("/:keyname/genre/:genre", keyinfosHandler.GetKeyInfoByKeynameAndGenre)
		}
		chat := protected.Group("/api/chat")
		{
			chat.GET("/user", chatHandler.GetChatSessions)
			chat.POST("/session/:session_id", chatHandler.CreateNewSession)
			chat.POST("/session/:session_id/send_msg", chatHandler.SendMessage)
			chat.PUT("/session/change_title", chatHandler.ChangeTitle)
			chat.DELETE("session/:session_id", chatHandler.DeleteChatSession)
			chat.GET("/session/:session_id", chatHandler.GetChatHistory)
			chat.POST("/stream", chatHandler.StreamChat)
		}

		prompt := protected.Group("/api/prompt")
		{
			prompt.POST("/", promptHandler.CreatePrompt)
			prompt.GET("/", promptHandler.GetAllPrompts)
			prompt.GET("/:id", promptHandler.GetPromptByID)
			prompt.GET("/keyname/:keyname", promptHandler.GetPromptByKeyname)
			prompt.PUT("/:id", promptHandler.UpdatePrompt)
			prompt.DELETE("/:id", promptHandler.DeletePrompt)
		}

		template := protected.Group("/api/template")
		{
			template.POST("/", templateHandler.CreateTemplate)
			template.GET("/", templateHandler.GetAllTemplates)
			template.GET("/:id", templateHandler.GetTemplateByID)
			template.GET("/keyname/:keyname", templateHandler.GetTemplateByKeyname)
			template.PUT("/:id", templateHandler.UpdateTemplate)
			template.DELETE("/:id", templateHandler.DeleteTemplate)
		}
	}

	return r
}
