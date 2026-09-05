package models

import (
	"time"
)

//nolint:tagliatelle
type RssContent struct {
	ID         int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title      string     `gorm:"type:varchar(255);not null;comment:标题" json:"title"`
	Content    string     `gorm:"type:text;comment:内容" json:"content"`
	LLMResult  string     `gorm:"type:text;comment:LLM返回的内容" json:"llm_result"`
	Date       time.Time  `gorm:"type:timestamp;not null;index;comment:日期" json:"date"`
	Categories string     `gorm:"type:varchar(100);not null;index;comment:分类" json:"categories"`
	Source     string     `gorm:"type:varchar(100);not null;comment:源" json:"source"`
	Link       string     `gorm:"type:varchar(255);not null;comment:链接" json:"link"`
	MD5        string     `gorm:"type:varchar(64);not null;uniqueIndex:uni_md5;comment:标题的md5值" json:"md5"`
	TimeStay   int        `gorm:"type:integer;not null;default:0;comment:阅读时间" json:"time_stay"`
	Dgraph     string     `gorm:"type:text;comment:dgraph关系数据" json:"dgraph"`
	CreatedAt  time.Time  `gorm:"type:timestamp;not null" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"type:timestamp;not null" json:"updated_at"`
	DeletedAt  *time.Time `gorm:"type:timestamp" json:"deleted_at,omitempty"`
}

// TableName 指定带 schema 的表名
func (RssContent) TableName() string {
	return "rss_content"
}
