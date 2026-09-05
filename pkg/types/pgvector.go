package types

// PgVector PostgreSQL + pgvector 配置
type PgVector struct {
	Host            string  `yaml:"host" json:"host"`
	Port            int     `yaml:"port" json:"port"`
	User            string  `yaml:"user" json:"user"`
	Password        string  `yaml:"password" json:"password"`
	DBName          string  `yaml:"dbName" json:"dbName"`
	RssCollection   string  `yaml:"rssCollection" json:"rssCollection"`
	DedupCollection string  `yaml:"dedupCollection" json:"dedupCollection"`
	Dimension       int64   `yaml:"dimension" json:"dimension"`
	Score           float32 `yaml:"score" json:"score"`
}
