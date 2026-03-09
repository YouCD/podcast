package types

import "time"

type ChangeTitleReq struct {
	Message   []*MessageInfo `json:"message"  binding:"required"`
	SessionID string         `json:"session_id"  binding:"required"`
}
type MessageInfo struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRecordResp struct {
	SessionID string    `json:"session_id"` // 会话ID，用于区分不同的对话
	UserID    int       `json:"user_id"`    // 用户ID
	Title     string    `json:"title"`      // 会话标题，可选
	CreatedAt time.Time `json:"created_at"`
}
