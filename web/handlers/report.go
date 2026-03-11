package handlers

import (
	"context"
	"net/http"
	"podcast/internal/ai/report/daily"
	"podcast/internal/service"
	"podcast/pkg/types"
	"text/template"

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
}

// NewReportsHandler 创建报告处理器
func NewReportsHandler(reportService *service.ReportService, podcastCfg *types.Podcast) *ReportsHandler {
	return &ReportsHandler{reportService: reportService, podcastCfg: podcastCfg}
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
	rr, err := r.reportService.GetByID(ctx, cast.ToInt(c.Param("id")))
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	if rr.LLMResult == "" {
		go func() {
			c2, err := daily.New(c, r.podcastCfg)
			if err != nil {
				ErrorWithMessage(c, "Report not found")
				return
			}
			_, err = c2.Invoke(c, cast.ToInt(c.Param("id")))
			if err != nil {
				ErrorWithMessage(c, "Report not found")
				return
			}
		}()
		t := template.New("report")
		parse, err := t.Parse(waitHtml)
		if err != nil {
			ErrorWithMessage(c, "Report not found")
			return
		}
		value := ctx.Value("request_id")
		data := map[string]interface{}{
			"request_id": value,
		}

		var buf buffer.Buffer
		err = parse.Execute(&buf, data)
		if err != nil {
			ErrorWithMessage(c, "Report not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
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
	go func() {
		id := c.Request.Context().Value("request_id").(string)
		ctx := context.WithValue(context.Background(), "request_id", id)
		dailyReport, err := daily.New(ctx, r.podcastCfg)
		if err != nil {
			log.WithCtx(ctx).Errorf("创建每日报告处理器失败: %v", err.Error())
			return
		}

		_, err = dailyReport.Invoke(ctx, cast.ToInt(c.Param("id")))
		if err != nil {
			log.WithCtx(ctx).Errorf("Report generation failed: %v", err)
			return
		}
	}()

	t := template.New("report")
	parse, err := t.Parse(waitHtml)
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	value := c.Request.Context().Value("request_id")
	data := map[string]interface{}{
		"request_id": value,
	}

	var buf buffer.Buffer
	err = parse.Execute(&buf, data)
	if err != nil {
		ErrorWithMessage(c, "Report not found")
		return
	}
	c.Data(http.StatusAccepted, "text/html; charset=utf-8", buf.Bytes())
}
