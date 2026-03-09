package models

import "time"

//nolint:tagliatelle
type Report struct {
	ID             int        `gorm:"primaryKey;autoIncrement" json:"id"`
	TimeArray      string     `gorm:"type:varchar(255);not null;default:'';comment:日期范围" json:"time_array"`
	Question       string     `gorm:"type:varchar(255);not null;comment:问题" json:"question"`
	LLMResult      string     `gorm:"type:longtext;comment:LLM返回的内容" json:"llm_result"`
	Content        string     `gorm:"type:longtext;comment:报告内容" json:"content"`
	Genre          int        `gorm:"type:tinyint(2);not null;default:1;comment:报告类型 1:日报 2:周报 3:月报" json:"genre"`
	PodcastMP3URL  string     `gorm:"type:varchar(255);not null;default:'';comment:播客地址" json:"podcast_mp3_url"`
	PodcastContent string     `gorm:"type:longtext;comment:播客内容" json:"podcast_content"`
	CreatedAt      time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) on update CURRENT_TIMESTAMP(3)" json:"updated_at"`
	DeletedAt      *time.Time `gorm:"type:datetime(3)" json:"deleted_at,omitempty"`
}

func (Report) TableName() string {
	return "report"
}
