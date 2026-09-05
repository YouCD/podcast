package cmd

import (
	"context"
	"os"

	"podcast/config"
	"podcast/internal/ai/pgvector"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"podcast/pkg/dgraph"

	"github.com/spf13/cobra"
	"github.com/youcd/toolkit/log"
)

// 初始化命令
var initializationCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化系统",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "version" {
			return
		}
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
		log.WithCtx(ctx).Info("[初始化] 开始初始化系统")

		// 初始化表结构（使用已注入的 db）
		log.WithCtx(ctx).Info("[初始化] 创建数据库表结构")
		models.InitializationTable(ctx, loadedDB)

		// 初始化用户
		log.WithCtx(ctx).Info("[初始化] 创建默认用户")
		userDao := dao.NewUserDao(loadedDB)
		err := userDao.Create(ctx, &models.User{
			Name:     "admin",
			Password: "admin",
		}, loadedCfg.Global.Token)
		if err != nil {
			log.WithCtx(ctx).Error(err)
		}

		// 初始化dGraph
		log.WithCtx(ctx).Info("[初始化] 连接 DGraph 图数据库")
		initDgraph(ctx, config.Cfg.Database.Dgraph)

		// 初始化PgVector
		log.WithCtx(ctx).Info("[初始化] 连接 PostgreSQL 并创建向量表")
		initPgVector(ctx)

		log.WithCtx(ctx).Info("[初始化] 系统初始化完成")
	},
}

func initPgVector(ctx context.Context) {
	p := pgvector.NewPgVector(ctx, config.Cfg.Database.PostgreSQL.ToPgVector())
	defer p.Close(ctx)
	err := p.CreateCollection(ctx, loadedCfg.Database.PostgreSQL.DedupCollection)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return
	}
	log.WithCtx(ctx).Info("初始化PgVector成功")
}

func initDgraph(ctx context.Context, host string) {
	d, err := dgraph.New(host)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return
	}
	err = d.Init(ctx)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return
	}
	log.WithCtx(ctx).Info("初始化DGraph成功")
}
