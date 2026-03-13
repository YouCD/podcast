package dao

import (
	"context"
	"fmt"
	"time"

	"podcast/internal/database/models"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

//nolint:interfacebloat
type RssContentDao interface {
	// Create 创建新的Rss
	// 参数: post Rss结构体指针
	// 返回: 错误信息
	Create(ctx context.Context, post *models.RssContent) error

	// FindAll 获取所有Rss
	// 返回: Rss切片和错误信息
	FindAll(ctx context.Context) ([]*models.RssContent, error)

	// FindByID 根据ID查找Rss
	// 参数: id 帖子ID
	// 返回: Rss指针和错误信息
	FindByID(ctx context.Context, id string) (*models.RssContent, error)

	// FindByIDWithLLMHTML 根据ID查找Rss的LLMHTML字段
	// 参数: id 帖子ID
	// 返回: Rss指针和错误信息
	FindByIDWithLLMHTML(ctx context.Context, id string) (*models.RssContent, error)

	// UpdateByFields 更新Rss
	//nolint:modernize
	UpdateByFields(ctx context.Context, field FieldKv, data map[string]interface{}) error

	// Delete 删除Rss
	// 参数: post Rss结构体指针
	// 返回: 错误信息
	Delete(ctx context.Context, post *models.RssContent) error

	// FindAllCategories 获取所有分类
	// 返回: 分类字符串切片和错误信息
	FindAllCategories(ctx context.Context) ([]string, error)

	// FindByCategory24H 获取指定分类下最近24小时的内容
	// 参数: category 分类名称
	// 返回: Rss切片和错误信息
	FindByCategory24H(ctx context.Context, category string) ([]*models.RssContent, error)

	// Save 保存Rss
	// 参数: post Rss结构体指针
	// 返回: 错误信息
	Save(ctx context.Context, post *models.RssContent) error

	// FindByMD5 根据MD5查找Rss
	// 参数: md5字符串
	// 返回: Rss指针和错误信息
	FindByMD5(ctx context.Context, md5s ...string) ([]*models.RssContent, error)

	// FindBy24H 获取最近24小时的内容
	// 参数: category 分类名称
	// 返回: Rss切片和错误信息
	FindBy24H(ctx context.Context) ([]*models.RssContent, error)

	// FindRead24H 获取最近24小时已读的内容
	FindRead24H(ctx context.Context) ([]*models.RssContent, error)

	// FindByIDWithNextLLMHTML 根据当前的ID 查找 同类型的下一个 Rss的LLMHTML内容
	FindByIDWithNextLLMHTML(ctx context.Context, id, categories string, withCreatedAt bool) (*models.RssContent, error)

	DistinctCategories(ctx context.Context) ([]string, error)

	// FindByCategoryAndDate 获取指定分类和日期的新闻内容
	// 参数: category 分类名称, date 日期
	// 返回: Rss切片和错误信息
	FindByCategoryAndDate(ctx context.Context, category string, date time.Time) ([]*models.RssContent, error)

	// FindByTodayCategory 获取今日的分类
	FindByTodayCategory(ctx context.Context) ([]string, error)

	// FindByDateRange 根据日期范围查找RSS内容
	FindByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.RssContent, error)

	// FindByDateRangeCategories 根据日期范围、分类查找RSS内容
	FindByDateRangeCategories(ctx context.Context, startDate, endDate time.Time, category string) ([]*models.RssContent, error)

	// Today24HLowQuality 获取今日24小时低质量数量
	Today24HLowQuality(ctx context.Context) (int64, error)

	// Today24HRead 获取今日24小时阅读数量
	Today24HRead(ctx context.Context) (int64, error)

	// FindNotReadContent 获取未读的内容
	FindNotReadContent(ctx context.Context) ([]*models.RssContent, error)

	// BatchCreate 批量创建
	BatchCreate(ctx context.Context, posts ...*models.RssContent) error
}

type rssContentDao struct {
	db *gorm.DB
}

// NewRssContentDao 创建 RSS 内容 DAO（依赖注入 *gorm.DB）
func NewRssContentDao(db *gorm.DB) RssContentDao {
	return &rssContentDao{db: db}
}

func (dao *rssContentDao) FindNotReadContent(ctx context.Context) ([]*models.RssContent, error) {
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where(`time_stay=0 AND categories != "low_quality" AND llm_result!=""`).Order("date DESC").Scan(&posts).Error
	return posts, err
}

func (dao *rssContentDao) DistinctCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Distinct("categories").Scan(&categories).Error
	return categories, err
}

// FindByIDWithNextLLMHTML 根据当前的ID 查找 同类型的下一个 Rss的LLMHTML内容
func (dao *rssContentDao) FindByIDWithNextLLMHTML(ctx context.Context, id, categories string, withCreatedAt bool) (*models.RssContent, error) {
	// 获取当前ID的 Categories
	var err error
	var rss *models.RssContent
	if !withCreatedAt {
		time24H := getTime24HAgo()
		err = dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories = ? AND created_at >= ? AND time_stay=0 AND id!=?", categories, time24H, id).First(&rss).Error
	} else {
		err = dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories = ? AND time_stay=0 AND id!=?", categories, id).First(&rss).Error
	}

	if err != nil {
		return nil, err
	}
	return rss, nil
}

const twentyFourHours = 24 * time.Hour

func getTime24HAgo() time.Time {
	return time.Now().Add(-twentyFourHours)
}

func (dao *rssContentDao) FindRead24H(ctx context.Context) ([]*models.RssContent, error) {
	time24H := getTime24HAgo()
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("created_at >= ? AND time_stay>0 ", time24H).Order("updated_at DESC").Scan(&posts).Error
	return posts, err
}

// FindBy24H 获取最近24小时的内容
func (dao *rssContentDao) FindBy24H(ctx context.Context) ([]*models.RssContent, error) {
	time24H := getTime24HAgo()
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("created_at >= ? AND time_stay=0", time24H).Order("date DESC").Scan(&posts).Error
	// err := models.GetDb().Model(&models.RssContent{}).Find(&posts).Error
	return posts, err
}

// Create 创建新的Rss
// 参数: post Rss结构体指针
// 返回: 错误信息
func (dao *rssContentDao) Create(ctx context.Context, post *models.RssContent) error {
	result := dao.db.Model(&models.RssContent{}).WithContext(ctx).Create(post)
	if result.Error != nil {
		//nolint:all
		if mysqlErr, ok := result.Error.(*mysql.MySQLError); ok {
			switch mysqlErr.Number {
			case 1062: // MySQL中表示重复条目的代码
				return gorm.ErrDuplicatedKey
			default:
				return fmt.Errorf("create 数据库错误: %w", result.Error)
			}
		}
		return fmt.Errorf("create 数据库错误: %w", result.Error)
	}
	return nil
}

func (dao *rssContentDao) BatchCreate(ctx context.Context, posts ...*models.RssContent) error {
	if len(posts) == 0 {
		return nil
	}

	return dao.db.Model(&models.RssContent{}).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 单事务批量插入，减少事务提交开销
		// 配合 MySQL 的 bulk_insert_buffer_size 参数
		if err := tx.CreateInBatches(posts, 1000).Error; err != nil {
			// 错误处理...
			return err
		}
		return nil
	})
}

// FindAll 获取所有Rss
// 返回: Rss切片和错误信息
func (dao *rssContentDao) FindAll(ctx context.Context) ([]*models.RssContent, error) {
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Find(&posts).Error
	return posts, err
}

// FindByID 根据ID查找Rss
// 参数: id rssID
// 返回: Rss指针和错误信息
func (dao *rssContentDao) FindByID(ctx context.Context, id string) (*models.RssContent, error) {
	var post models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).First(&post, id).Error
	return &post, err
}

// FindByIDWithLLMHTML 根据ID查找Rss的LLMHTML字段
// 参数: id rssID
// 返回: Rss指针和错误信息
func (dao *rssContentDao) FindByIDWithLLMHTML(ctx context.Context, id string) (*models.RssContent, error) {
	var post models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).First(&post, id).Error
	return &post, err
}

// Save 保存Rss
// 参数: post Rss结构体指针
// 返回: 错误信息
func (dao *rssContentDao) Save(ctx context.Context, post *models.RssContent) error {
	return dao.db.Model(&models.RssContent{}).WithContext(ctx).Save(post).Error
}

type FieldKv struct {
	Field string
	Value interface{} //nolint:modernize
}

// UpdateByFields 更新Rss
//
//nolint:all
func (dao *rssContentDao) UpdateByFields(ctx context.Context, field FieldKv, data map[string]interface{}) error {
	return dao.db.Model(&models.RssContent{}).WithContext(ctx).Where(fmt.Sprintf("%s = ?", field.Field), field.Value).Updates(data).Error
}

// Delete 删除Rss
// 参数: post Rss结构体指针
// 返回: 错误信息
func (dao *rssContentDao) Delete(ctx context.Context, post *models.RssContent) error {
	return dao.db.Model(&models.RssContent{}).WithContext(ctx).Delete(post).Error
}

// FindAllCategories 获取所有分类
// 返回: 分类字符串切片和错误信息
func (dao *rssContentDao) FindAllCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Distinct().Where("categories!=?", "low_quality").Pluck("categories", &categories).Error
	return categories, err
}

// FindByCategory24H 获取指定分类下最近24小时的内容
// 参数: category 分类名称
// 返回: Rss切片和错误信息
func (dao *rssContentDao) FindByCategory24H(ctx context.Context, category string) ([]*models.RssContent, error) {
	time24H := getTime24HAgo()
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories = ? AND created_at >= ? AND time_stay=0 ", category, time24H).Scan(&posts).Error
	return posts, err
}

// FindByMD5 根据MD5查找Rss
// 参数: md5字符串
// 返回: Rss指针和错误信息
// 如果找不到匹配的记录，将返回gorm.ErrRecordNotFound错误
//
//nolint:all
func (dao *rssContentDao) FindByMD5(ctx context.Context, md5s ...string) (post []*models.RssContent, err error) {
	err = dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("md5 in ?", md5s).Scan(&post).Error
	return post, err
}

// FindByCategoryAndDate 获取指定分类和日期的新闻内容
// 参数: category 分类名称, date 日期
// 返回: Rss切片和错误信息
func (dao *rssContentDao) FindByCategoryAndDate(ctx context.Context, category string, date time.Time) ([]*models.RssContent, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories = ? AND date >= ? AND date < ?", category, startOfDay, endOfDay).Find(&posts).Error
	return posts, err
}

// FindByDateRange 根据日期范围查找RSS内容
// 参数: startDate 开始日期, endDate 结束日期
// 返回: Rss切片和错误信息
func (dao *rssContentDao) FindByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.RssContent, error) {
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("date >= ? AND date <= ? AND categories!=? ", startDate, endDate, "low_quality").Find(&posts).Error
	return posts, err
}

func (dao *rssContentDao) FindByDateRangeCategories(ctx context.Context, startDate, endDate time.Time, category string) ([]*models.RssContent, error) {
	var posts []*models.RssContent
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("date >= ? AND date <= ? AND categories!=? AND categories=?", startDate, endDate, "low_quality", category).Find(&posts).Error
	return posts, err
}

// FindByTodayCategory 获取今日的分类
func (dao *rssContentDao) FindByTodayCategory(ctx context.Context) ([]string, error) {
	end := time.Now()
	start := time.Now().Add(-24 * time.Hour)

	var categories []string
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Distinct().Where("categories!=? AND date >= ? AND date < ? ", "low_quality", start, end).Pluck("categories", &categories).Error

	return categories, err
}

// Today24HLowQuality 获取今日24小时低质量数量
func (dao *rssContentDao) Today24HLowQuality(ctx context.Context) (int64, error) {
	end := time.Now()
	start := time.Now().Add(-24 * time.Hour)

	var c int64
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories=? AND date >= ? AND date < ? ", "low_quality", start, end).Count(&c).Error

	return c, err
}

// Today24HRead 获取今日24小时阅读数量
func (dao *rssContentDao) Today24HRead(ctx context.Context) (int64, error) {
	end := time.Now()
	start := time.Now().Add(-24 * time.Hour)

	var c int64
	err := dao.db.Model(&models.RssContent{}).WithContext(ctx).Where("categories!=? AND date >= ? AND date < ? AND time_stay>0 ", "low_quality", start, end).Count(&c).Error

	return c, err
}
