package handlers

import (
	"net/http"

	"podcast/internal/ai/llm"
	"podcast/internal/database/models"
	"podcast/internal/service"
	"podcast/pkg/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ChatHandler 处理聊天相关的所有请求
// (原 ChatRecordsHandler)
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler 创建新的聊天处理器
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// GetChatHistory 根据会话ID获取聊天记录
// (原 GetChatRecordBySessionID)
// 功能：获取单个会话的完整历史（元数据 + 所有消息）
func (h *ChatHandler) GetChatHistory(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		ErrorWithMessage(c, "Session ID is required")
		return
	}

	// 1. 获取会话元信息，并进行权限验证
	session, err := h.chatService.GetSessionBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ErrorWithMessage(c, "Chat session not found")
		} else {
			ErrorWithMessage(c, err.Error())
		}
		return
	}

	userID, exists := c.Get("userID")
	if !exists || session.UserID != userID.(int) {
		ErrorWithMessage(c, "Permission denied")
		return
	}

	// 2. 获取该会话的所有消息
	messages, err := h.chatService.GetMessagesBySessionID(sessionID)
	if err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	// 3. 组合数据返回
	// 返回一个包含会话信息和消息列表的结构体
	response := gin.H{
		"session":  session,
		"messages": messages,
	}
	Success(c, response)
}

// GetChatSessions 根据用户ID获取聊天记录列表
// (原 GetChatRecordsByUserID)
// 功能：获取用户所有会话的元数据列表，不包含具体消息内容
func (h *ChatHandler) GetChatSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessions, err := h.chatService.GetSessionsByUserID(c.Request.Context(), userID.(int))
	if err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	Success(c, sessions)
}

// CreateNewSession 创建一个新的会话
// (原 UpsertChatRecordBySessionID 的“创建”部分)
// 功能：将“创建会话”逻辑独立出来，由后端生成唯一的 SessionID
func (h *ChatHandler) CreateNewSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		ErrorWithMessage(c, "User not authenticated")
		return
	}
	// 从请求参数中获取session_id
	sessionID := c.Param("session_id")
	if sessionID == "" {
		ErrorWithMessage(c, "Session ID is required")
		return
	}

	newSession := &models.ChatSession{
		SessionID: sessionID,
		UserID:    userID.(int),
		Title:     "新会话", // 默认标题
	}

	if err := h.chatService.CreateSession(c.Request.Context(), newSession); err != nil {
		ErrorWithMessage(c, "Failed to create session: "+err.Error())
		return
	}

	Success(c, newSession)
}

// DeleteChatSession 删除聊天记录
// (原 DeleteChatRecordBySessionID)
// 功能：删除会话，由于设置了 OnDelete:CASCADE，关联的消息会自动被删除
func (h *ChatHandler) DeleteChatSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		ErrorWithMessage(c, "Session ID is required")
		return
	}

	// 1. 权限验证
	existingRecord, err := h.chatService.GetSessionBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ErrorWithMessage(c, "Chat session not found")
		} else {
			ErrorWithMessage(c, err.Error())
		}
		return
	}

	userID, exists := c.Get("userID")
	if !exists || existingRecord.UserID != userID.(int) {
		ErrorWithMessage(c, "Permission denied")
		return
	}

	// 2. 执行删除操作
	if err := h.chatService.DeleteSession(c.Request.Context(), existingRecord.ID); err != nil {
		ErrorWithMessage(c, "Failed to delete session: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "Chat session deleted successfully"})
}

// ChangeTitle 修改会话标题
// (原 ChangeTitle)
// 功能：逻辑基本不变，但增加了权限验证，并使用新的DAO
func (h *ChatHandler) ChangeTitle(c *gin.Context) {
	var req types.ChangeTitleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, "Invalid request format: "+err.Error())
		return
	}

	// 1. 权限验证
	session, err := h.chatService.GetSessionBySessionID(c.Request.Context(), req.SessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ErrorWithMessage(c, "Chat session not found")
		} else {
			ErrorWithMessage(c, err.Error())
		}
		return
	}
	userID, exists := c.Get("userID")
	if !exists || session.UserID != userID.(int) {
		ErrorWithMessage(c, "Permission denied")
		return
	}

	// 2. 调用 LLM 生成标题
	pool := llm.NewLLMPool()
	llmInfo, err := pool.Get(c)
	if err != nil {
		ErrorWithMessage(c, "Failed to get LLM info: "+err.Error())
		return
	}
	defer pool.Put(c, llmInfo)

	title, err := llm.ConversationTitle(c, llmInfo, req.Message)
	if err != nil {
		ErrorWithMessage(c, "Failed to generate title: "+err.Error())
		return
	}

	// 3. 更新标题
	if err := h.chatService.UpdateSessionTitle(c.Request.Context(), req.SessionID, title); err != nil {
		ErrorWithMessage(c, "Failed to update title: "+err.Error())
		return
	}

	Success(c, gin.H{
		"title": title,
	})
}

// SendMessage 在指定会话中发送一条新消息
// (原 UpsertChatRecordBySessionID 的“更新”部分)
// 功能：这个方法取代了原来的“更新整个JSON”逻辑，改为追加单条消息
func (h *ChatHandler) SendMessage(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		ErrorWithMessage(c, "Session ID is required")
		return
	}

	// 1. 权限验证
	session, err := h.chatService.GetSessionBySessionID(c.Request.Context(), sessionID)
	if err != nil {
		ErrorWithMessage(c, "Chat session not found or invalid")
		return
	}
	userID, exists := c.Get("userID")
	if !exists || session.UserID != userID.(int) {
		ErrorWithMessage(c, "Permission denied")
		return
	}

	// 2. 解析请求体，现在请求体只包含单条消息
	var req struct {
		Role             string `json:"role" binding:"required"` // e.g., "user", "assistant"
		Content          string `json:"content"`
		UUID             string `json:"uuid" binding:"required"`
		ReasoningContent string `json:"reasoning_content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, "Invalid request format: "+err.Error())
		return
	}

	// 3. 创建新消息
	newMessage := &models.Message{
		SessionID:        sessionID,
		Role:             req.Role,
		Content:          req.Content,
		ReasoningContent: req.ReasoningContent,
		UUID:             req.UUID,
	}

	if err := h.chatService.CreateMessage(newMessage); err != nil {
		ErrorWithMessage(c, "Failed to save message: "+err.Error())
		return
	}

	// 4. (可选但推荐) 更新会话的 UpdatedAt 时间戳，使其在列表中置顶
	session.UpdatedAt = newMessage.CreatedAt
	h.chatService.UpdateSession(c.Request.Context(), session)

	Success(c, newMessage)
}
