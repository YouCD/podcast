package models

import "time"

//nolint:tagliatelle
type KeyInfo struct {
	ID        int        `gorm:"primaryKey;autoIncrement" json:"id"`
	KeyName   string     `gorm:"type:varchar(100);not null;comment:键名" json:"key_name"`
	Data      string     `gorm:"type:longtext;comment:数据字段" json:"data"`
	Genre     int        `gorm:"type:int(11);not null;default:0;comment:类型1:LLM分析各类prompt,2:html模板,3:LLM报表分析prompt" json:"genre"`
	CreatedAt time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) on update CURRENT_TIMESTAMP(3)" json:"updated_at"`
	DeletedAt *time.Time `gorm:"type:datetime(3)" json:"deleted_at,omitempty"`
}

func (KeyInfo) TableName() string {
	return "key_info"
}
