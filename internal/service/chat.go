package service

import (
	"context"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
)

// ChatService 聊天业务服务（会话 + 消息）
type ChatService struct {
	sessionsDAO *dao.ChatSessionsDAO
	messagesDAO *dao.MessagesDAO
}

// NewChatService 创建聊天服务
func NewChatService(sessionsDAO *dao.ChatSessionsDAO, messagesDAO *dao.MessagesDAO) *ChatService {
	return &ChatService{sessionsDAO: sessionsDAO, messagesDAO: messagesDAO}
}

// GetSessionBySessionID 根据 sessionID 获取会话
func (s *ChatService) GetSessionBySessionID(ctx context.Context, sessionID string) (*models.ChatSession, error) {
	return s.sessionsDAO.GetSessionBySessionID(ctx, sessionID)
}

// GetSessionsByUserID 获取用户所有会话列表
func (s *ChatService) GetSessionsByUserID(ctx context.Context, userID int) ([]models.ChatSession, error) {
	return s.sessionsDAO.GetSessionsByUserID(ctx, userID)
}

// CreateSession 创建会话
func (s *ChatService) CreateSession(ctx context.Context, session *models.ChatSession) error {
	return s.sessionsDAO.CreateSession(ctx, session)
}

// UpdateSession 更新会话
func (s *ChatService) UpdateSession(ctx context.Context, session *models.ChatSession) error {
	return s.sessionsDAO.UpdateSession(ctx, session)
}

// UpdateSessionTitle 更新会话标题
func (s *ChatService) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	return s.sessionsDAO.UpdateSessionTitle(ctx, sessionID, title)
}

// DeleteSession 删除会话（会级联删除消息）
func (s *ChatService) DeleteSession(ctx context.Context, id uint) error {
	return s.sessionsDAO.DeleteSession(ctx, id)
}

// GetMessagesBySessionID 获取会话下所有消息
func (s *ChatService) GetMessagesBySessionID(sessionID string) ([]*models.Message, error) {
	return s.messagesDAO.GetMessagesBySessionID(sessionID)
}

// CreateMessage 创建一条消息
func (s *ChatService) CreateMessage(message *models.Message) error {
	return s.messagesDAO.CreateMessage(message)
}
