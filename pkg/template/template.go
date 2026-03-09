package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"podcast/config"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"podcast/pkg/types"
	"strings"
	"text/template"

	"github.com/youcd/toolkit/log"
)

var (
	h *HTMLTemplateManager
)

type HTMLTemplateManager struct {
}

func GetHTMLTemplateManager() *HTMLTemplateManager {
	return h
}

func (h *HTMLTemplateManager) RenderFromJSON(ctx context.Context, rss *models.RssContent) (string, error) {
	//nolint:modernize
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(rss.LLMResult), &jsonData)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %w", err)
	}
	jsonData["Title"] = rss.Title
	jsonData["Date"] = rss.Date.Format("2006-01-02 15:04:05")
	jsonData["Link"] = rss.Link
	jsonData["Source"] = strings.ReplaceAll(rss.Source, " ", "")
	jsonData["Categories"] = rss.Categories

	// 获取模板,模板名称与分类一致
	genre, _ := dao.NewKeyInfoDao(models.GetDb()).FindByKeynameAndGenre(ctx, rss.Categories, 2)
	if len([]rune(rss.Content)) < config.Cfg.Global.ContentLen || genre == nil {
		log.WithCtx(ctx).Debugw("renderLlmResult", "templateName", "默认模板", "md5", rss.MD5, "jsonData", jsonData)
		return h.renderLlmResult(ctx, rss)
	}
	log.WithCtx(ctx).Debugw("renderFromJSON", "templateName", rss.Categories, "md5", rss.MD5, "jsonData", jsonData)
	var buf bytes.Buffer
	tpl := template.Must(template.New(rss.Categories).Funcs(funcMap).Parse(genre.Data))
	if err := tpl.Execute(&buf, jsonData); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return buf.String(), nil
}

func (h *HTMLTemplateManager) renderLlmResult(ctx context.Context, rss *models.RssContent) (string, error) {
	tpl := template.Must(template.New("defaultTmpl").Funcs(funcMap).Parse(defaultTmpl))

	//nolint:all
	type llmResult struct {
		Title      string `json:"Title"`
		Date       string `json:"Date"`
		Link       string `json:"Link"`
		Source     string `json:"Source"`
		Categories string `json:"Categories"`
		types.LLMResult
	}
	var buf bytes.Buffer
	var result llmResult
	err := json.Unmarshal([]byte(rss.LLMResult), &result)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error: %w", err)
	}
	result.Title = rss.Title
	result.Date = rss.Date.Format("2006-01-02 15:04:05")
	result.Link = rss.Link
	result.Source = strings.ReplaceAll(rss.Source, " ", "")
	result.Categories = rss.Categories

	err = tpl.Execute(&buf, result)
	if err != nil {
		log.WithCtx(ctx).Warnw("default_renderLlmResult", "err", err)
		return rss.LLMResult, nil
	}
	return buf.String(), nil
}
