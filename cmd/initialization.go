package cmd

import (
	"context"
	"os"
	"podcast/config"
	"podcast/internal/ai/milvus"
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
		// 初始化表结构（使用已注入的 db）
		models.InitializationTable(ctx, loadedDB)
		//  初始化用户
		userDao := dao.NewUserDao(loadedDB)
		err := userDao.Create(ctx, &models.User{
			Name:     "admin",
			Password: "admin",
		}, loadedCfg.Global.Token)
		if err != nil {
			log.WithCtx(ctx).Error(err)
		}
		//  初始化dGraph
		initDgraph(ctx)
		//  初始化Milvus
		initMilvus(ctx)
	},
}

func initMilvus(ctx context.Context) {
	m := milvus.New(ctx)
	defer m.Close(ctx)
	err := m.CreateDedupCollection(ctx, loadedCfg.Database.Milvus.DedupCollection)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return
	}
	log.WithCtx(ctx).Info("初始化Milvus成功")
}

func initDgraph(ctx context.Context) {
	d, err := dgraph.New()
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
