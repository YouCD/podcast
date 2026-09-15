package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"text/template"

	"podcast/internal/ai/llm"
	"podcast/internal/ai/report/daily"
	"podcast/internal/service"
	"podcast/pkg/types"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/youcd/toolkit/log"
	"go.uber.org/zap/buffer"
)

// ReportResponse 定义不包含LLMResult的Report响应结构
//
//nolint:all
type ReportResponse struct {
	ID            int    `json:"id"`
	Question      string `json:"question"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	TimeArray     string `json:"time_array"`
	PodcastMP3URL string `json:"podcast_mp3_url"`
}
type ReportsHandler struct {
	reportService *service.ReportService
	podcastCfg    *types.Podcast
	llmPool       *llm.LLMPool
	generating    sync.Map // reportID(int) -> struct{}，记录生成中的报告，防止并发重复触发
}

// NewReportsHandler 创建报告处理器
func NewReportsHandler(reportService *service.ReportService, podcastCfg *types.Podcast, llmPool *llm.LLMPool) *ReportsHandler {
	return &ReportsHandler{reportService: reportService, podcastCfg: podcastCfg, llmPool: llmPool, generating: sync.Map{}}
}

// GetReports 获取所有report列表，但不包含LLMResult
func (r *ReportsHandler) GetReports(c *gin.Context) {
	// 检查是否有genre参数
	genreStr := c.Query("genre")
	reportList, err := r.reportService.GetAllByGenre(c.Request.Context(), cast.ToInt(genreStr))
	if err != nil {
		ErrorWithMessage(c, "Failed to fetch reports")
		return
	}

	// 转换为不包含LLMResult的响应结构
	response := make([]ReportResponse, len(reportList))
	for i, report := range reportList {
		response[i] = ReportResponse{
			ID:            report.ID,
			Question:      report.Question,
			CreatedAt:     report.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:     report.UpdatedAt.Format("2006-01-02 15:04:05"),
			TimeArray:     report.TimeArray,
			PodcastMP3URL: report.PodcastMP3URL,
		}
	}

	Success(c, response)
}

// GetLLMResultByID 根据ID获取指定report的LLMResult
func (r *ReportsHandler) GetLLMResultByID(c *gin.Context) {
	ctx := c.Request.Context()
	id := cast.ToInt(c.Param("id"))
	rr, err := r.reportService.GetByID(ctx, id)
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	if rr.LLMResult == "" {
		r.startGeneration(ctx, id)
		page, err := renderWaitPage(ctx.Value("request_id"))
		if err != nil {
			ErrorWithMessage(c, "render wait page failed")
			return
		}
		c.Data(http.StatusAccepted, "text/html; charset=utf-8", page)
		return
	}
	htmlContent := modifyHtml5(rr)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// GetReportDetailByID 根据ID获取指定report详情
func (r *ReportsHandler) GetReportDetailByID(c *gin.Context) {
	rr, err := r.reportService.GetByID(c.Request.Context(), cast.ToInt(c.Param("id")))
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	htmlContent := modifyHtml5(rr)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// PlayByID 播放指定ID的音频
func (r *ReportsHandler) PlayByID(c *gin.Context) {
	ctx := c.Request.Context()
	rr, err := r.reportService.GetByID(ctx, cast.ToInt(c.Param("id")))
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	log.WithCtx(ctx).Info(rr.PodcastMP3URL)
	filePath := r.reportService.PodcastFilePath(rr)
	// 设置正确的Content-Type
	c.Header("Content-Type", "audio/mpeg")
	c.File(filePath)
}

// GenDailyReport 生成每日报告
func (r *ReportsHandler) GenDailyReport(c *gin.Context) {
	r.startGeneration(c.Request.Context(), cast.ToInt(c.Param("id")))
	page, err := renderWaitPage(c.Request.Context().Value("request_id"))
	if err != nil {
		ErrorWithMessage(c, "render wait page failed")
		return
	}
	c.Data(http.StatusAccepted, "text/html; charset=utf-8", page)
}

// startGeneration 后台触发报告生成，同一报告并发时只触发一次
func (r *ReportsHandler) startGeneration(ctx context.Context, id int) {
	if _, loaded := r.generating.LoadOrStore(id, struct{}{}); loaded {
		return
	}
	go func() {
		defer r.generating.Delete(id)
		// 脱离请求取消，保留 request_id 等值，避免响应写完后后台任务被取消
		bctx := context.WithoutCancel(ctx)
		c2, err := daily.New(bctx, r.podcastCfg, r.llmPool)
		if err != nil {
			log.WithCtx(bctx).Errorf("创建报告生成器失败: %v", err)
			return
		}
		if _, err = c2.Invoke(bctx, id); err != nil {
			log.WithCtx(bctx).Errorf("报告生成失败: %v", err)
		}
	}()
}

// renderWaitPage 渲染报告生成等待页
func renderWaitPage(requestID any) ([]byte, error) {
	t := template.New("report")
	parse, err := t.Parse(waitHtml)
	if err != nil {
		return nil, fmt.Errorf("parse wait html: %w", err)
	}
	var buf buffer.Buffer
	if err = parse.Execute(&buf, map[string]any{"request_id": requestID}); err != nil {
		return nil, fmt.Errorf("execute wait html: %w", err)
	}
	return buf.Bytes(), nil
}
