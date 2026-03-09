package service

import (
	"context"
	"path"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
)

// ReportService 报告业务服务
type ReportService struct {
	reportDao  dao.ReportDao
	podcastDir string
}

// NewReportService 创建报告服务，podcastDir 用于播放时解析本地文件路径
func NewReportService(reportDao dao.ReportDao, podcastDir string) *ReportService {
	return &ReportService{reportDao: reportDao, podcastDir: podcastDir}
}

// GetAllByGenre 按 genre 获取报告列表（不含 LLMResult）
func (s *ReportService) GetAllByGenre(ctx context.Context, genre int) ([]*models.Report, error) {
	return s.reportDao.GetAllReportsByGenre(ctx, genre)
}

// GetByID 根据 ID 获取报告
func (s *ReportService) GetByID(ctx context.Context, id int) (*models.Report, error) {
	return s.reportDao.GetReportByID(ctx, id)
}

// PodcastFilePath 返回报告对应播客文件的本地路径
func (s *ReportService) PodcastFilePath(report *models.Report) string {
	if s.podcastDir == "" || report == nil || report.PodcastMP3URL == "" {
		return ""
	}
	return path.Join(s.podcastDir, report.PodcastMP3URL)
}
