package service

import (
	"context"

	"podcast/internal/database/models"
)

const promptGenre = 1

// PromptService Prompt 业务服务（Genre=1 的 KeyInfo）
type PromptService struct {
	keyInfo *KeyInfoService
}

// NewPromptService 创建 Prompt 服务
func NewPromptService(keyInfo *KeyInfoService) *PromptService {
	return &PromptService{keyInfo: keyInfo}
}

// Create 创建 Prompt（强制 Genre=1）
func (s *PromptService) Create(ctx context.Context, prompt *models.KeyInfo) error {
	prompt.Genre = promptGenre
	return s.keyInfo.Create(ctx, prompt)
}

// FindAll 获取所有 Prompt
func (s *PromptService) FindAll(ctx context.Context) ([]*models.KeyInfo, error) {
	return s.keyInfo.FindByGenre(ctx, promptGenre)
}

// FindByID 根据 ID 获取，并校验为 Prompt 类型
func (s *PromptService) FindByID(ctx context.Context, id int) (*models.KeyInfo, error) {
	prompt, err := s.keyInfo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if prompt.Genre != promptGenre {
		return nil, errNotPrompt
	}
	return prompt, nil
}

// FindByKeyname 根据 keyname 获取
func (s *PromptService) FindByKeyname(ctx context.Context, keyname string) (*models.KeyInfo, error) {
	return s.keyInfo.FindByKeynameAndGenre(ctx, keyname, promptGenre)
}

// Update 更新（强制 Genre=1）
func (s *PromptService) Update(ctx context.Context, prompt *models.KeyInfo) error {
	prompt.Genre = promptGenre
	return s.keyInfo.Update(ctx, prompt)
}

// Delete 删除（需先校验为 Prompt 类型）
func (s *PromptService) Delete(ctx context.Context, id int) error {
	prompt, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return s.keyInfo.Delete(ctx, prompt)
}
