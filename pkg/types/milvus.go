package types

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
