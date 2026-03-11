package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"podcast/pkg/types"
	"syscall"
	"time"

	"podcast/config"
	"podcast/internal/app"
	"podcast/internal/database/models"
	"podcast/pkg/cron"
	"podcast/web/routes"

	"github.com/spf13/cobra"
	"github.com/youcd/toolkit/log"
	"gorm.io/gorm"
)

var (
	configPath string
	Name       = "Podcast"
	// 供 Run 使用的 DI 依赖（由 PersistentPreRun 赋值）
	loadedCfg *config.Config
	loadedDB  *gorm.DB
)

func init() {
	rootCmd.AddCommand(versionCmd, setCmd, initializationCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "file", "f", "config.yaml", "config file path")
}

var rootCmd = &cobra.Command{
	Use:  Name,
	Long: fmt.Sprintf("%s is an application that aggregates RSS feeds, processes them with LLM and provides a web interface", Name),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "version" {
			return
		}
		// 初始化数据库连接
		cfg, err := config.LoadAppConfig(configPath)
		if err != nil {
			log.WithCtx(cmd.Context()).Error(err)
			os.Exit(1)
		}
		db, err := models.Init(cfg)
		if err != nil {
			log.WithCtx(cmd.Context()).Error(err)
			os.Exit(1)
		}
		loadedCfg = cfg
		loadedDB = db
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// 启动Rss定时任务
		log.WithCtx(cmd.Context()).Info("starting RSS cron job...")
		go func() {
			cron.StartRSSCronJob(ctx, &types.RagConfig{
				Milvus:     config.Cfg.Database.Milvus,
				Embedding:  config.Cfg.Vector.Embedding,
				DgraphHost: config.Cfg.Database.Dgraph,
			}, config.Cfg.RSS)
			// 启动报告定时任务
			cron.StartReportJob(ctx, config.Cfg.Report, config.Cfg.Podcast)
		}()
		// 设置路由（依赖注入：cfg + db -> container -> router）
		container, err := app.New(ctx, loadedCfg, loadedDB)
		if err != nil {
			log.WithCtx(cmd.Context()).Fatalf("failed to create container: %v", err)
		}
		r := routes.SetupRouter(container)
		// 在 goroutine 中启动服务
		srv := &http.Server{
			Addr:    loadedCfg.Global.HostPort,
			Handler: r,
		}
		go func() {
			log.WithCtx(cmd.Context()).Infof("server listening at: %s", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.WithCtx(cmd.Context()).Panicf("listen: %s\n", err)
			}
		}()
		// 等待中断信号
		<-ctx.Done()

		log.WithCtx(cmd.Context()).Info("shutting down server")

		// 优雅关闭服务器
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			if err := container.Close(ctx); err != nil {
				log.WithCtx(ctx).Error(err)
			}
			cancel()
		}()
		if err := srv.Shutdown(ctxShutdown); err != nil {
			log.WithCtx(ctx).Panicf("Server forced to shutdown: %v", err)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.WithCtx(rootCmd.Context()).Error(err)
		os.Exit(1)
	}
}
