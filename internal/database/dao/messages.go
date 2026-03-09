package dao

import (
	"podcast/internal/database/models"

	"gorm.io/gorm"
)

type MessagesDAO struct {
	db *gorm.DB
}

// NewMessagesDAO 创建 MessagesDAO（依赖注入 *gorm.DB）
func NewMessagesDAO(db *gorm.DB) *MessagesDAO {
	return &MessagesDAO{db: db}
}

func (d *MessagesDAO) GetMessagesBySessionID(sessionID string) ([]*models.Message, error) {
	var messages []*models.Message
	err := d.db.Model(&models.Message{}).Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (d *MessagesDAO) CreateMessage(message *models.Message) error {
	return d.db.Model(&models.Message{}).Create(message).Error
}
