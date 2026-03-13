package handlers

import (
	"strconv"

	"podcast/internal/database/models"
	"podcast/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/youcd/toolkit/log"
)

// KeyInfoHandler 定义KeyInfo处理器
type KeyInfoHandler struct {
	keyInfoService *service.KeyInfoService
}

// NewKeyInfoHandler 创建 KeyInfo 处理器
func NewKeyInfoHandler(keyInfoService *service.KeyInfoService) *KeyInfoHandler {
	return &KeyInfoHandler{keyInfoService: keyInfoService}
}

// CreateKeyInfo 创建KeyInfo
func (k *KeyInfoHandler) CreateKeyInfo(c *gin.Context) {
	var keyInfo models.KeyInfo
	if err := c.ShouldBindJSON(&keyInfo); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	if err := k.keyInfoService.Create(c.Request.Context(), &keyInfo); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("创建KeyInfo失败: %v", err)
		ErrorWithMessage(c, "创建KeyInfo失败")
		return
	}

	Success(c, keyInfo)
}

// GetAllKeyInfos 获取所有KeyInfos
func (k *KeyInfoHandler) GetAllKeyInfos(c *gin.Context) {
	keyInfos, err := k.keyInfoService.FindAll(c.Request.Context())
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取KeyInfo列表失败: %v", err)
		ErrorWithMessage(c, "获取KeyInfo列表失败")
		return
	}

	Success(c, keyInfos)
}

// GetKeyInfoByID 根据ID获取KeyInfo
func (k *KeyInfoHandler) GetKeyInfoByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	keyInfo, err := k.keyInfoService.FindByID(c.Request.Context(), id)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取KeyInfo失败: %v", err)
		ErrorWithMessage(c, "KeyInfo未找到")
		return
	}

	Success(c, keyInfo)
}

// GetKeyInfoByKeyName 根据KeyName获取KeyInfo
func (k *KeyInfoHandler) GetKeyInfoByKeyName(c *gin.Context) {
	keyName := c.Param("key_name")

	keyInfo, err := k.keyInfoService.FindByKeyName(c.Request.Context(), keyName)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取KeyInfo失败: %v", err)
		ErrorWithMessage(c, "KeyInfo未找到")
		return
	}

	Success(c, keyInfo)
}

// UpdateKeyInfo 更新KeyInfo
func (k *KeyInfoHandler) UpdateKeyInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	var keyInfo models.KeyInfo
	if err := c.ShouldBindJSON(&keyInfo); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	// 确保ID一致
	keyInfo.ID = id

	if err := k.keyInfoService.Update(c.Request.Context(), &keyInfo); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("更新KeyInfo失败: %v", err)
		ErrorWithMessage(c, "更新KeyInfo失败")
		return
	}

	Success(c, keyInfo)
}

// DeleteKeyInfo 删除KeyInfo
func (k *KeyInfoHandler) DeleteKeyInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorWithMessage(c, "无效的ID")
		return
	}

	keyInfo, err := k.keyInfoService.FindByID(c.Request.Context(), id)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取KeyInfo失败: %v", err)
		ErrorWithMessage(c, "KeyInfo未找到")
		return
	}

	if err := k.keyInfoService.Delete(c.Request.Context(), keyInfo); err != nil {
		log.WithCtx(c.Request.Context()).Errorf("删除KeyInfo失败: %v", err)
		ErrorWithMessage(c, "删除KeyInfo失败")
		return
	}

	Success(c, gin.H{"message": "KeyInfo删除成功"})
}

// GetKeyInfosByGenre 根据Genre获取KeyInfo列表
func (k *KeyInfoHandler) GetKeyInfosByGenre(c *gin.Context) {
	genre, err := strconv.Atoi(c.Param("genre"))
	if err != nil {
		ErrorWithMessage(c, "无效的Genre")
		return
	}

	keyInfos, err := k.keyInfoService.FindByGenre(c.Request.Context(), genre)
	if err != nil {
		log.WithCtx(c.Request.Context()).Errorf("获取KeyInfo列表失败: %v", err)
		ErrorWithMessage(c, "获取KeyInfo列表失败")
		return
	}

	Success(c, keyInfos)
}

// GetKeyInfoByKeynameAndGenre 根据Keyname和Genre获取KeyInfo
func (k *KeyInfoHandler) GetKeyInfoByKeynameAndGenre(c *gin.Context) {
	keyname := c.Param("keyname")
	genre, err := strconv.Atoi(c.Param("genre"))
	if err != nil {
		ErrorWithMessage(c, "无效的Genre")
		return
	}
	ctx := c.Request.Context()
	keyInfo, err := k.keyInfoService.FindByKeynameAndGenre(ctx, keyname, genre)
	if err != nil {
		log.WithCtx(ctx).Errorf("获取KeyInfo失败: %v", err)
		ErrorWithMessage(c, "KeyInfo未找到")
		return
	}

	Success(c, keyInfo)
}
