package models

import "time"

// ChatSession 存储会话的元信息
type ChatSession struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID string     `gorm:"type:varchar(255);not null;uniqueIndex;comment:会话ID" json:"session_id"`
	UserID    int        `gorm:"not null;index;comment:用户ID" json:"user_id"`
	Title     string     `gorm:"type:varchar(255);comment:会话标题" json:"title"`
	CreatedAt time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) on update CURRENT_TIMESTAMP(3)" json:"updated_at"`
	DeletedAt *time.Time `gorm:"type:datetime(3)" json:"deleted_at,omitempty"`
}

func (ChatSession) TableName() string {
	return "chat_session"
}

// Message 存储每一条具体的消息
type Message struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID        string `gorm:"type:varchar(255);not null;index;comment:会话ID" json:"session_id"` // 外键
	UUID             string `gorm:"type:varchar(255);not null;comment:消息ID" json:"uuid"`
	ReasoningContent string `gorm:"type:text;comment:推理内容" json:"reasoning_content"`
	Role             string `gorm:"type:varchar(50);not null;comment:角色" json:"role"` // 'user', 'assistant', 'system'
	Content          string `gorm:"type:text;not null;comment:消息内容" json:"content"`   // 消息内容
	// 新增字段：计划数据（JSON格式）
	Plan string `gorm:"type:longtext;comment:计划数据" json:"plan"`
	// 新增字段：步骤数据（JSON格式）
	Steps string `gorm:"type:longtext;comment:步骤数据" json:"steps"`
	// 新增字段：思考内容
	ThinkContent string    `gorm:"type:longtext;comment:思考内容" json:"think_content"`
	CreatedAt    time.Time `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (Message) TableName() string {
	return "message"
}
