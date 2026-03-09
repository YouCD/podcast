package dao

import (
	"context"
	"fmt"
	"podcast/internal/database/models"
	"time"

	"gorm.io/gorm"
)

// ReportDao 定义 Report 数据访问接口
type ReportDao interface {
	// GetAllReports 获取所有报告列表（不包括 LLMResult）
	GetAllReports(ctx context.Context) ([]*models.Report, error)

	// GetReportByID 根据ID获取指定报告
	GetReportByID(ctx context.Context, id int) (*models.Report, error)

	// Create 创建新的Repo
	Create(ctx context.Context, post *models.Report) error

	// GetAllReportsByGenre 根据类型获取报告列表
	GetAllReportsByGenre(ctx context.Context, genre int) ([]*models.Report, error)

	// GetRepoByQuestion 根据问题获取报告
	GetRepoByQuestion(ctx context.Context, question string) (*models.Report, error)

	// UpdateByFields 根据ID更新报告的指定字段
	//nolint:modernize
	UpdateByFields(ctx context.Context, field FieldKv, updates map[string]interface{}) error
}

// reportDao 实现 ReportDao 接口
type reportDao struct {
	db *gorm.DB
}

// NewReportDao 创建 ReportDao（依赖注入 *gorm.DB）
func NewReportDao(db *gorm.DB) ReportDao {
	return &reportDao{db: db}
}

// GetAllReports 获取所有报告列表（不包括 LLMResult）
func (r *reportDao) GetAllReports(ctx context.Context) ([]*models.Report, error) {
	var reportList []*models.Report
	// 当前月的开始日期
	now := time.Now().Add(-time.Hour * 24 * 30)

	// 构造本月的第一天，小时、分钟、秒和纳秒都设为0
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	err := r.db.Model(&models.Report{}).WithContext(ctx).Where("created_at >= ?", firstDayOfMonth).Order("created_at DESC").Scan(&reportList).Error
	return reportList, err
}

func (r *reportDao) Create(ctx context.Context, post *models.Report) error {
	// 实现创建报告逻辑
	return r.db.Model(&models.Report{}).WithContext(ctx).Create(post).Error
}

// GetAllReportsByGenre 根据类型获取报告列表
func (r *reportDao) GetAllReportsByGenre(ctx context.Context, genre int) ([]*models.Report, error) {
	var reportList []*models.Report
	// 当前月的开始日期
	now := time.Now().Add(-time.Hour * 24 * 30)

	// 构造本月的第一天，小时、分钟、秒和纳秒都设为0
	firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	err := r.db.Model(&models.Report{}).WithContext(ctx).Where("genre = ? AND created_at >= ?", genre, firstDayOfMonth).Order("created_at DESC").Scan(&reportList).Error
	return reportList, err
}

// GetReportByID 根据ID获取指定报告
func (r *reportDao) GetReportByID(ctx context.Context, id int) (*models.Report, error) {
	var report models.Report
	err := r.db.Model(&models.Report{}).WithContext(ctx).First(&report, id).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}
func (r *reportDao) GetRepoByQuestion(ctx context.Context, question string) (*models.Report, error) {
	var report *models.Report
	err := r.db.Model(&models.Report{}).WithContext(ctx).Where("question = ?", question).Scan(&report).Error
	if err != nil {
		return nil, err
	}
	return report, nil
}

// UpdateByFields 根据ID更新报告的指定字段
//
//nolint:all
func (r *reportDao) UpdateByFields(ctx context.Context, field FieldKv, updates map[string]interface{}) error {
	err := r.db.Model(&models.Report{}).WithContext(ctx).Where(fmt.Sprintf("%s = ?", field.Field), field.Value).Updates(updates).Error
	return err
}
