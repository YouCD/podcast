package cmd

import (
	"os"

	"podcast/config"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/spf13/cobra"
	"github.com/youcd/toolkit/log"
)

var (
	user string
	pwd  string
)

func init() {
	setCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "用户名")
	setCmd.PersistentFlags().StringVarP(&pwd, "password", "p", "", "密码")
}

// setCmd 设置账户名称命令
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "设置账户名称",
	Long:  `设置系统管理员账户的用户名`,
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
		if user == "" {
			log.WithCtx(cmd.Context()).Error("请指定新的账户名称")
			os.Exit(1)
		}

		var isChange bool

		// 检查用户是否存在
		userDao := dao.NewUserDao(loadedDB)
		userInfo, err := userDao.FindByName(cmd.Context(), user)
		if err != nil {
			log.WithCtx(cmd.Context()).Errorf("账户名称 '%s' 不存在", user)
			os.Exit(1)
		}
		if userInfo != nil {
			isChange = true
		}

		if isChange {
			// 更新用户名
			oldName := userInfo.Name
			userInfo.Name = user
			if pwd != "" {
				userInfo.Password = pwd
			}

			err = userDao.Update(cmd.Context(), userInfo, loadedCfg.Global.Token)
			if err != nil {
				log.WithCtx(cmd.Context()).Errorf("更新账户信息失败: %v", err)
				os.Exit(1)
			}
			if oldName != user {
				log.WithCtx(cmd.Context()).Infof("账户名称已从 '%s' 成功更改为 '%s'", oldName, user)
			}
			return
		}

		if pwd == "" {
			log.WithCtx(cmd.Context()).Error("请指定新的账户密码")
			os.Exit(1)
		}
		err = userDao.Create(cmd.Context(), &models.User{Name: user, Password: pwd}, loadedCfg.Global.Token)
		if err != nil {
			log.WithCtx(cmd.Context()).Errorf("创建账户失败: %v", err)
			os.Exit(1)
		}
		log.WithCtx(cmd.Context()).Infof("账户已创建成功: %s", user)
	},
}
