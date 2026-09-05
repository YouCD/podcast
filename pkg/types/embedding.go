package types

import "time"

//nolint:tagliatelle
type Embedding struct {
	APIKey   string `yaml:"apiKey" json:"apiKey"`
	Model    string `yaml:"model" json:"model"`
	BaseURL  string `yaml:"baseURL" json:"baseURL"`
	Provider string `yaml:"provider" json:"provider"` // dashscope | ollama，为空时默认 dashscope
	Dimension int64  `yaml:"dimension" json:"dimension"`
}

// RagConfig RAG 引擎配置
type RagConfig struct {
	PgVector   *PgVector
	Embedding  *Embedding
	DgraphHost string
	LLMTimeout time.Duration // 单条 LLM 分析超时时间
}
