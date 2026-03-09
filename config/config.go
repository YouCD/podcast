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
	Mysql  *Mysql `json:"mysql" yaml:"mysql"`
	Dgraph string `json:"dgraph" yaml:"dgraph"`
	Milvus Milvus `yaml:"milvus" json:"milvus"`
}

// Global 全局配置（导出供 DI 使用）
type Global struct {
	LogLevel   string `json:"logLevel" yaml:"logLevel" validate:"required,oneof=debug info warn error panic fatal"`
	LogFile    string `json:"logFile" yaml:"logFile" validate:"omitempty"`
	HostPort   string `json:"hostPort" yaml:"hostPort" validate:"required"`
	ContentLen int    `json:"contentLen" yaml:"contentLen"`
	Token      string `json:"token" yaml:"token"`
	PodcastDir string `json:"podcastDir" yaml:"podcastDir"`
}

// Milvus 向量库配置（导出供 DI 使用）
type Milvus struct {
	Endpoint        string  `yaml:"endpoint" json:"endpoint"`
	APIKey          string  `yaml:"apiKey" json:"apiKey"`
	DBName          string  `yaml:"dbName" json:"dbName"`
	RssCollection   string  `yaml:"rssCollection" json:"rssCollection"`
	DedupCollection string  `yaml:"dedupCollection" json:"dedupCollection"`
	Dimension       int64   `yaml:"dimension" json:"dimension"`
	Score           float32 `yaml:"score" json:"score"`
}

type Mysql struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DBName   string `json:"dbName" yaml:"dbname"`
}

type byteDance struct {
	AppID       string `json:"appID" yaml:"appID"`
	AccessToken string `json:"accessToken" yaml:"accessToken"`
}

//nolint:tagliatelle
type LLM struct {
	ApiKey  string `json:"apiKey,omitempty" yaml:"apiKey"`
	BaseURL string `json:"baseURL,omitempty" yaml:"baseURL"`
	Model   string `json:"model,omitempty" yaml:"model"`
	Prompt  string `json:"prompt,omitempty" yaml:"prompt"`
}
type vector struct {
	Embedding embedding `yaml:"embedding" json:"embedding"`
}

//nolint:tagliatelle
type embedding struct {
	APIKey  string `yaml:"apiKey" json:"apiKey"`
	BaseURL string `yaml:"baseURL" json:"baseURL"`
	Model   string `yaml:"model" json:"model"`
}

type report struct {
	Schedule string `yaml:"schedule" json:"schedule"`
	Topic    string `yaml:"topic" json:"topic"`
}

// Config 应用配置（导出供依赖注入使用）
//
//nolint:all
type Config struct {
	Database  *Database             `json:"database" yaml:"database"`
	RSS       []*types.RSSSource    `json:"rss" yaml:"rss"`
	Global    *Global               `json:"global" yaml:"global"`
	LLM       []*LLM                `json:"LLM" yaml:"LLM"`
	Vector    vector                `json:"vector" yaml:"vector"`
	Report    []*report             `json:"report" yaml:"report"`
	MCPProxy  map[string]*types.Mcp `json:"mcpProxy" yaml:"mcpProxy"`
	ByteDance byteDance             `json:"byteDance" yaml:"byteDance"`
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
	log.Init(logConfig)
	log.SetLogLevel(Cfg.Global.LogLevel)
	log.WithCtx(context.Background()).Debug("日志级别： ", Cfg.Global.LogLevel)

	return Cfg, nil
}
