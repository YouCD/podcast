package handlers

import (
	"errors"
	"podcast/internal/database/models"
	"podcast/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/youcd/toolkit/log"
)

// PromptHandler 定义Prompt处理器
type PromptHandler struct {
	promptService *service.PromptService
}

// NewPromptHandler 创建Prompt处理器实例
func NewPromptHandler(promptService *service.PromptService) *PromptHandler {
	return &PromptHandler{promptService: promptService}
}

// CreatePrompt 创建Prompt
func (h *PromptHandler) CreatePrompt(c *gin.Context) {
	var prompt models.KeyInfo
	if err := c.ShouldBindJSON(&prompt); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	// Genre 由 PromptService.Create 固定为 1
	if err := h.promptService.Create(c.Request.Context(), &prompt); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("创建Prompt失败: %v", err)
		ErrorWithMessage(c, "创建Prompt失败")
		return
	}

	Success(c, prompt)
}

// GetAllPrompts 获取所有Prompts
func (h *PromptHandler) GetAllPrompts(c *gin.Context) {
	prompts, err := h.promptService.FindAll(c.Request.Context())
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Prompt列表失败: %v", err)
		ErrorWithMessage(c, "获取Prompt列表失败")
		return
	}

	Success(c, prompts)
}

// GetPromptByID 根据ID获取Prompt
func (h *PromptHandler) GetPromptByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	prompt, err := h.promptService.FindByID(c.Request.Context(), id)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Prompt失败: %v", err)
		if errors.Is(err, service.ErrNotPrompt) {
			ErrorWithMessage(c, "不是有效的Prompt记录")
		} else {
			ErrorWithMessage(c, "Prompt未找到")
		}
		return
	}

	Success(c, prompt)
}

// GetPromptByKeyname 根据Keyname获取Prompt
func (h *PromptHandler) GetPromptByKeyname(c *gin.Context) {
	keyname := c.Param("keyname")

	prompt, err := h.promptService.FindByKeyname(c.Request.Context(), keyname)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Prompt失败: %v", err)
		ErrorWithMessage(c, "Prompt未找到")
		return
	}

	Success(c, prompt)
}

// UpdatePrompt 更新Prompt
func (h *PromptHandler) UpdatePrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	var prompt models.KeyInfo
	if err := c.ShouldBindJSON(&prompt); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	prompt.ID = id
	// Genre 由 PromptService 在 Update 内固定为 1
	if err := h.promptService.Update(c.Request.Context(), &prompt); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("更新Prompt失败: %v", err)
		ErrorWithMessage(c, "更新Prompt失败")
		return
	}

	Success(c, prompt)
}

// DeletePrompt 删除Prompt
func (h *PromptHandler) DeletePrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	if err := h.promptService.Delete(c.Request.Context(), id); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("删除Prompt失败: %v", err)
		if errors.Is(err, service.ErrNotPrompt) {
			ErrorWithMessage(c, "不是有效的Prompt记录")
		} else {
			ErrorWithMessage(c, "删除Prompt失败")
		}
		return
	}
	Success(c, gin.H{"message": "Prompt删除成功"})
}

// TemplateHandler 定义Template处理器
type TemplateHandler struct {
	templateService *service.TemplateService
}

// NewTemplateHandler 创建Template处理器实例
func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// CreateTemplate 创建Template
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var template models.KeyInfo
	if err := c.ShouldBindJSON(&template); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	// 设置Genre为2表示html模板
	template.Genre = 2

	if err := h.templateService.Create(c.Request.Context(), &template); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("创建Template失败: %v", err)
		ErrorWithMessage(c, "创建Template失败")
		return
	}

	Success(c, template)
}

// GetAllTemplates 获取所有Templates
func (h *TemplateHandler) GetAllTemplates(c *gin.Context) {
	templates, err := h.templateService.FindAll(c.Request.Context())
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Template列表失败: %v", err)
		ErrorWithMessage(c, "获取Template列表失败")
		return
	}

	Success(c, templates)
}

// GetTemplateByID 根据ID获取Template
func (h *TemplateHandler) GetTemplateByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	template, err := h.templateService.FindByID(c.Request.Context(), id)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Template失败: %v", err)
		if errors.Is(err, service.ErrNotTemplate) {
			ErrorWithMessage(c, "不是有效的Template记录")
		} else {
			ErrorWithMessage(c, "Template未找到")
		}
		return
	}

	Success(c, template)
}

// GetTemplateByKeyname 根据Keyname获取Template
func (h *TemplateHandler) GetTemplateByKeyname(c *gin.Context) {
	keyname := c.Param("keyname")

	template, err := h.templateService.FindByKeyname(c.Request.Context(), keyname)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取Template失败: %v", err)
		ErrorWithMessage(c, "Template未找到")
		return
	}

	Success(c, template)
}

// UpdateTemplate 更新Template
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	var template models.KeyInfo
	if err := c.ShouldBindJSON(&template); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	template.ID = id
	if err := h.templateService.Update(c.Request.Context(), &template); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("更新Template失败: %v", err)
		ErrorWithMessage(c, "更新Template失败")
		return
	}

	Success(c, template)
}

// DeleteTemplate 删除Template
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	if err := h.templateService.Delete(c.Request.Context(), id); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("删除Template失败: %v", err)
		if errors.Is(err, service.ErrNotTemplate) {
			ErrorWithMessage(c, "不是有效的Template记录")
		} else {
			ErrorWithMessage(c, "删除Template失败")
		}
		return
	}

	Success(c, gin.H{"message": "Template删除成功"})
}
