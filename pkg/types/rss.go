package types

import "time"

// RSSSource 定义RSS源的结构
type RSSSource struct {
	Name    string `json:"name" yaml:"name"`
	URL     string `json:"url" yaml:"url"`
	Disable bool   `json:"disable" yaml:"disable"`
}

// RSSItem 用于表示解析后的RSS项目
type RSSItem struct {
	Title      string
	Content    string
	Date       time.Time
	Categories string
	Source     string
	Link       string
	MD5        string
	LLMResult  string
	Dgraph     string
	Vector     []float32 // 向量数据，用于去重缓存
}
