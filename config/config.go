package config

import (
	"context"
	"fmt"
	"os"

	"podcast/pkg/types"

	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"github.com/youcd/toolkit/file"
	"github.com/youcd/toolkit/log"
)

var Cfg *Config

// Database 数据库配置（导出供 DI 使用）
type Database struct {
	PostgreSQL *PostgreSQL `json:"postgresql" yaml:"postgresql"`
	Dgraph     string      `json:"dgraph" yaml:"dgraph"`
}

// PostgreSQL PostgreSQL 数据库配置
type PostgreSQL struct {
	Host            string  `json:"host" yaml:"host"`
	Port            int     `json:"port" yaml:"port"`
	User            string  `json:"user" yaml:"user"`
	Password        string  `json:"password" yaml:"password"`
	DBName          string  `json:"dbName" yaml:"dbName"`
	RssCollection   string  `yaml:"rssCollection" json:"rssCollection"`
	DedupCollection string  `yaml:"dedupCollection" json:"dedupCollection"`
	Dimension       int64   `yaml:"dimension" json:"dimension"`
	Score           float32 `yaml:"score" json:"score"`
}

// ToPgVector 转换为 PgVector 配置
func (p *PostgreSQL) ToPgVector() *types.PgVector {
	return &types.PgVector{
		Host:            p.Host,
		Port:            p.Port,
		User:            p.User,
		Password:        p.Password,
		DBName:          p.DBName,
		RssCollection:   p.RssCollection,
		DedupCollection: p.DedupCollection,
		Dimension:       p.Dimension,
		Score:           p.Score,
	}
}

// Global 全局配置（导出供 DI 使用）
type Global struct {
	LogLevel       string `json:"logLevel" yaml:"logLevel" validate:"required,oneof=debug info warn error panic fatal"`
	LogFile        string `json:"logFile" yaml:"logFile" validate:"omitempty"`
	HostPort       string `json:"hostPort" yaml:"hostPort" validate:"required"`
	ContentLen     int    `json:"contentLen" yaml:"contentLen"`
	Token          string `json:"token" yaml:"token"`
	MaxConcurrency int    `json:"maxConcurrency" yaml:"maxConcurrency"` // LLM 最大并发数
	LLMTimeout     int    `json:"llmTimeout" yaml:"llmTimeout"`         // 单条 LLM 分析超时（秒），0 表示默认 120 秒
}

type vector struct {
	Embedding *types.Embedding `yaml:"embedding" json:"embedding"`
}

// Config 应用配置（导出供依赖注入使用）
//
//nolint:all
type Config struct {
	Database *Database             `json:"database" yaml:"database"`
	RSS      []*types.RSSSource    `json:"rss" yaml:"rss"`
	Global   *Global               `json:"global" yaml:"global"`
	LLM      []*types.LLMInfo      `json:"LLM" yaml:"LLM"`
	Vector   vector                `json:"vector" yaml:"vector"`
	Report   []*types.Report       `json:"report" yaml:"report"`
	MCPProxy map[string]*types.Mcp `json:"mcpProxy" yaml:"mcpProxy"`
	Podcast  *types.Podcast        `json:"podcast" yaml:"podcast"`
}

// LoadAppConfig 使用 viper 从YAML文件加载应用配置
func LoadAppConfig(fileName string) (*Config, error) {
	if !file.Exists(fileName) {
		fmt.Printf("配置文件不存在: %s\n", fileName)
		os.Exit(1)
	}
	viper.SetConfigType("yaml")
	viper.SetConfigFile(fileName)
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 将配置绑定到结构体
	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	logConfig := &log.Config{
		Stdout: true,
	}
	if Cfg.Global.LogFile != "" {
		logConfig.LumberjackCfg = &lumberjack.Logger{
			Filename: Cfg.Global.LogFile,
		}
	}

	if Cfg.Global.ContentLen == 0 {
		Cfg.Global.ContentLen = 1400
	}
	if Cfg.Global.LLMTimeout == 0 {
		Cfg.Global.LLMTimeout = 120
	}
	// 设置默认向量表名
	if Cfg.Database.PostgreSQL.DedupCollection == "" {
		Cfg.Database.PostgreSQL.DedupCollection = "dedup_title"
	}
	if Cfg.Database.PostgreSQL.RssCollection == "" {
		Cfg.Database.PostgreSQL.RssCollection = "rss_vectors"
	}
	if Cfg.Database.PostgreSQL.Dimension == 0 {
		Cfg.Database.PostgreSQL.Dimension = 1024
	}
	if Cfg.Database.PostgreSQL.Score == 0 {
		Cfg.Database.PostgreSQL.Score = 0.78
	}
	log.Init(logConfig)
	log.SetLogLevel(Cfg.Global.LogLevel)
	log.WithCtx(context.Background()).Debug("日志级别： ", Cfg.Global.LogLevel)
	// 设置 embedding 模型维度与 PostgreSQL 向量维度一致
	Cfg.Vector.Embedding.Dimension = Cfg.Database.PostgreSQL.Dimension
	return Cfg, nil
}
