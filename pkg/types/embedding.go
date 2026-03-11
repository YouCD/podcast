package types

//nolint:tagliatelle
type Embedding struct {
	APIKey    string `yaml:"apiKey" json:"apiKey"`
	Model     string `yaml:"model" json:"model"`
	Dimension int64  `yaml:"dimension" json:"dimension"`
}

// RagConfig RAG 引擎配置
type RagConfig struct {
	Milvus     *Milvus
	Embedding  *Embedding
	DgraphHost string
}
