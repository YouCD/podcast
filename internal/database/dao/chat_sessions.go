package dao

import (
	"context"
	"podcast/internal/database/models"

	"gorm.io/gorm"
)

type ChatSessionsDAO struct {
	db *gorm.DB
}

// NewChatSessionsDAO 创建 ChatSessionsDAO（依赖注入 *gorm.DB）
func NewChatSessionsDAO(db *gorm.DB) *ChatSessionsDAO {
	return &ChatSessionsDAO{db: db}
}

func (d *ChatSessionsDAO) GetSessionBySessionID(ctx context.Context, sessionID string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := d.db.Model(&models.ChatSession{}).WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (d *ChatSessionsDAO) GetSessionsByUserID(ctx context.Context, userID int) ([]models.ChatSession, error) {
	var sessions []models.ChatSession
	// 优化：只查询列表需要的字段，不加载关联的 Messages
	err := d.db.Model(&models.ChatSession{}).WithContext(ctx).Select("id", "session_id", "user_id", "title", "created_at", "updated_at").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (d *ChatSessionsDAO) CreateSession(ctx context.Context, session *models.ChatSession) error {
	return d.db.Model(&models.ChatSession{}).WithContext(ctx).Create(session).Error
}

func (d *ChatSessionsDAO) UpdateSession(ctx context.Context, session *models.ChatSession) error {
	return d.db.Model(&models.ChatSession{}).WithContext(ctx).Where("session_id = ?", session.SessionID).Save(session).Error
}

func (d *ChatSessionsDAO) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	return d.db.Model(&models.ChatSession{}).WithContext(ctx).Where("session_id = ?", sessionID).Update("title", title).Error
}

func (d *ChatSessionsDAO) DeleteSession(ctx context.Context, id uint) error {
	// GORM 的软删除
	return d.db.Model(&models.ChatSession{}).WithContext(ctx).Delete(&models.ChatSession{}, id).Error
}
