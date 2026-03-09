package service

import (
	"context"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
)

// RssService RSS/Feed 业务服务
type RssService struct {
	rssDao dao.RssContentDao
}

// NewRssService 创建 RSS 服务
func NewRssService(rssDao dao.RssContentDao) *RssService {
	return &RssService{rssDao: rssDao}
}

// FindAll 获取所有 RSS 内容
func (s *RssService) FindAll(ctx context.Context) ([]*models.RssContent, error) {
	return s.rssDao.FindAll(ctx)
}

// FindByID 根据 ID 获取
func (s *RssService) FindByID(ctx context.Context, id string) (*models.RssContent, error) {
	return s.rssDao.FindByID(ctx, id)
}

// FindByIDWithLLMHTML 根据 ID 获取（含 LLM 内容）
func (s *RssService) FindByIDWithLLMHTML(ctx context.Context, id string) (*models.RssContent, error) {
	return s.rssDao.FindByIDWithLLMHTML(ctx, id)
}

// FindByIDWithNextLLMHTML 根据当前 ID 获取同分类下一条
func (s *RssService) FindByIDWithNextLLMHTML(ctx context.Context, id, categories string, withCreatedAt bool) (*models.RssContent, error) {
	return s.rssDao.FindByIDWithNextLLMHTML(ctx, id, categories, withCreatedAt)
}

// Save 保存（创建或更新）
func (s *RssService) Save(ctx context.Context, post *models.RssContent) error {
	return s.rssDao.Save(ctx, post)
}

// UpdateByFields 按条件更新指定字段
func (s *RssService) UpdateByFields(ctx context.Context, field dao.FieldKv, data map[string]interface{}) error {
	return s.rssDao.UpdateByFields(ctx, field, data)
}

// FindByMD5 根据 MD5 查询
func (s *RssService) FindByMD5(ctx context.Context, md5 string) ([]*models.RssContent, error) {
	return s.rssDao.FindByMD5(ctx, md5)
}

// FindAllCategories 获取所有分类
func (s *RssService) FindAllCategories(ctx context.Context) ([]string, error) {
	return s.rssDao.FindAllCategories(ctx)
}

// FindByCategory24H 指定分类最近 24 小时
func (s *RssService) FindByCategory24H(ctx context.Context, category string) ([]*models.RssContent, error) {
	return s.rssDao.FindByCategory24H(ctx, category)
}

// FindBy24H 最近 24 小时内容
func (s *RssService) FindBy24H(ctx context.Context) ([]*models.RssContent, error) {
	return s.rssDao.FindBy24H(ctx)
}

// FindRead24H 最近 24 小时已读
func (s *RssService) FindRead24H(ctx context.Context) ([]*models.RssContent, error) {
	return s.rssDao.FindRead24H(ctx)
}

// FindNotReadContent 未读内容
func (s *RssService) FindNotReadContent(ctx context.Context) ([]*models.RssContent, error) {
	return s.rssDao.FindNotReadContent(ctx)
}

// Today24HLowQuality 今日 24 小时低质量数量
func (s *RssService) Today24HLowQuality(ctx context.Context) (int64, error) {
	return s.rssDao.Today24HLowQuality(ctx)
}

// Today24HRead 今日 24 小时已读数量
func (s *RssService) Today24HRead(ctx context.Context) (int64, error) {
	return s.rssDao.Today24HRead(ctx)
}

// UpdateTimeStay 更新阅读时长（业务封装：先查再更新）
func (s *RssService) UpdateTimeStay(ctx context.Context, md5 string, timeStay int) error {
	list, err := s.rssDao.FindByMD5(ctx, md5)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return errRssNotFound
	}
	return s.rssDao.UpdateByFields(ctx, dao.FieldKv{Field: "md5", Value: md5}, map[string]interface{}{
		"time_stay": timeStay,
	})
}
