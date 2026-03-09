package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"podcast/pkg/dgraph"
	"podcast/pkg/types"

	"podcast/internal/database/models"
	"podcast/internal/service"
	"podcast/pkg/template"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"
	"github.com/youcd/toolkit/log"
)

var _ RssServiceProvider = (*RssHandler)(nil)

// RssServiceProvider 供测试或替换注入
type RssServiceProvider interface {
	GetRssService() *service.RssService
}

// RssHandler RSS/Feed 处理器
type RssHandler struct {
	rssService *service.RssService
}

// NewRssHandler 创建 RSS 处理器
func NewRssHandler(rssService *service.RssService) *RssHandler {
	return &RssHandler{rssService: rssService}
}

// GetRssService 实现 RssServiceProvider
func (r *RssHandler) GetRssService() *service.RssService { return r.rssService }

// TimeStayRequest 定义接收time_stay和md5字段的请求结构体
//
//nolint:all
type TimeStayRequest struct {
	TimeStay int    `json:"time_stay"`
	MD5      string `json:"md5"`
}

// RssContentResponse Rss响应结构体，不包含LLMHTML字段
//
//nolint:all
type RssContentResponse struct {
	ID         int               `json:"id"`
	Title      string            `json:"title"`
	Date       time.Time         `json:"date"`
	Categories string            `json:"categories"`
	Source     string            `json:"source"`
	Score      int               `json:"score"`
	Link       string            `json:"link"`
	MD5        string            `json:"md5"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	LLMHTML    string            `json:"llm_html,omitempty"`
	Dgraph     *types.DgraphResp `json:"dgraph,omitempty"`
}

// 将CommunityPost转换为CommunityPostResponse
func toRssResponse(rss *models.RssContent, htmlStr string) *RssContentResponse {
	return &RssContentResponse{
		ID:         rss.ID,
		Title:      rss.Title,
		Date:       rss.Date,
		Categories: rss.Categories,
		Source:     rss.Source,
		Link:       rss.Link,
		MD5:        rss.MD5,
		CreatedAt:  rss.CreatedAt,
		UpdatedAt:  rss.UpdatedAt,
		LLMHTML:    htmlStr,
	}
}

// 将CommunityPost切片转换为CommunityPostResponse切片
func toCommunityPostResponseSlice(posts []*models.RssContent) []*RssContentResponse {
	responses := make([]*RssContentResponse, len(posts))
	for i, post := range posts {
		responses[i] = toRssResponse(post, "")
	}
	return responses
}

// GetRss24H 获取所有最近24小时已读的内容
func (r *RssHandler) GetRss24H(c *gin.Context) {
	// 1. 查找对应的文章记录
	content, err := r.rssService.FindRead24H(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, "内容不存在")
		return
	}
	Success(c, toCommunityPostResponseSlice(content))
}

// HandleTimeStayAndMD5 处理time_stay和md5字段的接口
func (r *RssHandler) HandleTimeStayAndMD5(c *gin.Context) {
	var req TimeStayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}
	if req.MD5 == "" || req.TimeStay < 0 {
		ErrorWithMessage(c, "MD5 is required and TimeStay must be non-negative")
		return
	}
	err := r.rssService.UpdateTimeStay(c.Request.Context(), req.MD5, req.TimeStay)
	if err != nil {
		if err == service.ErrRssNotFound {
			ErrorWithMessage(c, fmt.Sprintf("Content with MD5 %s not found.", req.MD5))
		} else {
			ErrorWithMessage(c, "Failed to update database record.")
		}
		return
	}

	Success(c, gin.H{
		"message": "success",
	})
}

// GetRssContent 获取所有Rss内容
func (r *RssHandler) GetRssContent(c *gin.Context) {
	posts, err := r.rssService.FindAll(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, "Failed to fetch posts")
		return
	}

	Success(c, toCommunityPostResponseSlice(posts))
}

// GetRssContentByID 根据ID获取Rss内容
func (r *RssHandler) GetRssContentByID(c *gin.Context) {
	id := c.Param("id")
	post, err := r.rssService.FindByID(c.Request.Context(), id)
	if err != nil {
		ErrorWithMessage(c, "Post not found")
		return
	}

	Success(c, toRssResponse(post, ""))
}

// GetLLMHTMLByID 根据ID获取Rss内容的LLMHTML字段
func (r *RssHandler) GetLLMHTMLByID(c *gin.Context) {
	id := c.Param("id")
	var withCreatedAt bool
	value, b := c.GetQuery("not_read")
	if b {
		withCreatedAt = strings.ToLower(value) == "true"
	}
	rss, err := r.rssService.FindByIDWithLLMHTML(c.Request.Context(), id)
	if err != nil {
		ErrorWithMessage(c, "Post not found")
		return
	}

	htmlStr, err := template.GetHTMLTemplateManager().RenderFromJSON(c, rss)
	if err != nil {
		Success(c, gin.H{"current": rss, "next": nil})
		return
	}

	current := toRssResponse(rss, htmlStr)
	resp, err := dgraphResp(c, rss.Dgraph)
	if err != nil {
		log.WithCtx(c).Errorw("dgraph", "error", err)
	}
	if resp != nil {
		current.Dgraph = resp
		log.WithCtx(c).Infow("dgraph", "node", len(resp.Nodes), "edge", len(resp.Edges))
	}
	next, err := r.rssService.FindByIDWithNextLLMHTML(c.Request.Context(), id, rss.Categories, withCreatedAt)
	if err != nil {
		Success(c, gin.H{"current": current, "next": nil})
		return
	}

	if next != nil {
		nextHtmlStr, err := template.GetHTMLTemplateManager().RenderFromJSON(c, next)
		if err == nil {
			nextItem := toRssResponse(next, nextHtmlStr)

			resp, err = dgraphResp(c, next.Dgraph)
			if err != nil {
				log.WithCtx(c).Warnw("dgraph", "error", err)
			}
			if resp != nil {
				nextItem.Dgraph = resp
			}

			Success(c, gin.H{"current": current, "next": nextItem})
			return
		}
	}
	Success(c, gin.H{"current": current, "next": nil})
}

func dgraphResp(ctx context.Context, dgraphStr string) (*types.DgraphResp, error) {
	var payload types.DgraphPayload
	err := json.Unmarshal([]byte(dgraphStr), &payload)
	if err != nil {
		log.WithCtx(ctx).Errorw("dgraph", "error", err)
		return nil, fmt.Errorf("解析Dgraph数据失败: %v", err)
	}

	d, err := dgraph.New()
	if err != nil {
		log.WithCtx(ctx).Errorw("dgraph", "error", err)
		return nil, fmt.Errorf("创建Dgraph客户端失败: %v", err)
	}
	var nodes []string
	for _, node := range payload.Set {
		one, err := d.QueryOne(ctx, node.Name)
		if err != nil {
			log.WithCtx(ctx).Errorw("dgraph", "error", err)
			continue
		}
		nodes = append(nodes, one.Uid)
	}
	uids := strings.Join(nodes, ",")
	dql := fmt.Sprintf(`{
  q(func: uid(%s)) {
    uid
    dgraph.type
    expand(_all_) {
      uid
      name
      dgraph.type
      expand(_all_) {
        uid
        name
        dgraph.type
      }
    }
  }
}
`, uids)

	return d.Query(ctx, dql)
}

// UpdateRssContent 更新Rss内容
func (r *RssHandler) UpdateRssContent(c *gin.Context) {
	id := c.Param("id")
	post, err := r.rssService.FindByID(c.Request.Context(), id)
	if err != nil {
		ErrorWithMessage(c, "Post not found")
		return
	}

	if err := c.ShouldBindJSON(post); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	if err := r.rssService.Save(c.Request.Context(), post); err != nil {
		ErrorWithMessage(c, "Failed to update post")
		return
	}

	Success(c, toRssResponse(post, ""))
}

// GetAllCategories 获取所有分类
func (r *RssHandler) GetAllCategories(c *gin.Context) {
	categories, err := r.rssService.FindAllCategories(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, "Failed to fetch categories")
		return
	}

	Success(c, categories)
}

// GetRssByCategory24H 获取指定分类下最近24小时的内容
func (r *RssHandler) GetRssByCategory24H(c *gin.Context) {
	category := c.Param("category")
	posts, err := r.rssService.FindByCategory24H(c.Request.Context(), category)
	if err != nil {
		ErrorWithMessage(c, "Failed to fetch posts for category: "+category)
		return
	}
	result := make([]RssContentResponse, 0)
	for _, post := range posts {
		if strings.Contains(post.LLMResult, "low_quality") {
			continue
		}
		result = append(result, *toRssResponse(post, ""))
	}
	log.WithCtx(c).Debugw("GetRssByCategory24H", "Categories", category, "Content", result)
	Success(c, result)
}

// ExportRss24H 导出最近24小时的RSS内容
func (r *RssHandler) ExportRss24H(c *gin.Context) {
	posts, err := r.rssService.FindBy24H(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, "Failed to fetch posts")
		return
	}

	// 生成RSS feed
	rssFeed, err := generateRSSFeed(posts)
	if err != nil {
		ErrorWithMessage(c, "Failed to generate RSS feed")
		return
	}

	c.Data(http.StatusOK, "application/xml", []byte(rssFeed))
}

// generateRSSFeed 生成RSS feed XML
func generateRSSFeed(posts []*models.RssContent) (string, error) {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Podcast RSS Feed",
		Link:        &feeds.Link{Href: "https://rss.youcd.online"},
		Description: "Latest podcast content",
		Author:      &feeds.Author{Name: "Podcast Team"},
		Created:     now,
	}

	feed.Items = []*feeds.Item{}
	for _, post := range posts {
		item := &feeds.Item{
			Id:      post.MD5,
			Title:   post.Title,
			Link:    &feeds.Link{Href: post.Link},
			Content: post.Content,
			Created: post.Date,
			Updated: post.UpdatedAt,
		}
		feed.Items = append(feed.Items, item)
	}

	rss, err := feed.ToRss()
	if err != nil {
		return "", fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	return rss, nil
}

// Status 状态
func (r *RssHandler) Status(c *gin.Context) {
	ctx := c.Request.Context()
	count, _ := r.rssService.Today24HLowQuality(ctx)
	readCount, _ := r.rssService.Today24HRead(ctx)
	Success(c, gin.H{"low_quality_count": count, "read_count": readCount})
}

func (r *RssHandler) NotRead(c *gin.Context) {
	content, err := r.rssService.FindNotReadContent(c.Request.Context())
	if err != nil {
		ErrorWithMessage(c, "内容不存在")
		return
	}
	Success(c, toCommunityPostResponseSlice(content))
}
