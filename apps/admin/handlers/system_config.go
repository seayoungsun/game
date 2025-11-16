package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

// GetSystemConfigs 获取系统配置列表
func GetSystemConfigs(c *gin.Context) {
	var configs []models.SystemConfig

	query := database.DB.Model(&models.SystemConfig{})

	// 按分组筛选
	if groupName := c.Query("group_name"); groupName != "" {
		query = query.Where("group_name = ?", groupName)
	}

	// 只获取公开配置
	if c.Query("public") == "true" {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Order("group_name ASC, config_key ASC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": configs,
	})
}

// GetSystemConfig 获取单个系统配置
func GetSystemConfig(c *gin.Context) {
	configKey := c.Param("key")

	var config models.SystemConfig
	if err := database.DB.Where("config_key = ?", configKey).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "配置不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取配置失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": config,
	})
}

// CreateSystemConfig 创建系统配置
func CreateSystemConfig(c *gin.Context) {
	var req struct {
		ConfigKey   string `json:"config_key" binding:"required"`
		ConfigValue string `json:"config_value"`
		ConfigType  string `json:"config_type" binding:"required"`
		GroupName   string `json:"group_name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 检查配置键是否已存在
	var existConfig models.SystemConfig
	if err := database.DB.Where("config_key = ?", req.ConfigKey).First(&existConfig).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "配置键已存在",
		})
		return
	}

	if req.GroupName == "" {
		req.GroupName = "default"
	}
	if req.ConfigType == "" {
		req.ConfigType = "string"
	}

	now := time.Now().Unix()
	config := models.SystemConfig{
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		GroupName:   req.GroupName,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := database.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    config,
	})
}

// UpdateSystemConfig 更新系统配置
func UpdateSystemConfig(c *gin.Context) {
	configKey := c.Param("key")

	var req struct {
		ConfigValue interface{} `json:"config_value"` // 使用 interface{} 接受字符串或数字
		Description string      `json:"description"`
		IsPublic    *bool       `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var config models.SystemConfig
	if err := database.DB.Where("config_key = ?", configKey).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "配置不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取配置失败: " + err.Error(),
			})
		}
		return
	}

	// 更新字段
	if req.ConfigValue != nil {
		// 将 config_value 转换为字符串
		var configValueStr string
		switch v := req.ConfigValue.(type) {
		case string:
			configValueStr = v
		case float64:
			// JSON 数字通常解析为 float64
			configValueStr = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			configValueStr = strconv.Itoa(v)
		case int64:
			configValueStr = strconv.FormatInt(v, 10)
		case bool:
			if v {
				configValueStr = "true"
			} else {
				configValueStr = "false"
			}
		default:
			configValueStr = fmt.Sprintf("%v", v)
		}
		config.ConfigValue = configValueStr
	}
	if req.Description != "" {
		config.Description = req.Description
	}
	if req.IsPublic != nil {
		config.IsPublic = *req.IsPublic
	}
	config.UpdatedAt = time.Now().Unix()

	if err := database.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    config,
	})
}

// DeleteSystemConfig 删除系统配置
func DeleteSystemConfig(c *gin.Context) {
	configKey := c.Param("key")

	if err := database.DB.Where("config_key = ?", configKey).Delete(&models.SystemConfig{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetSystemConfigGroups 获取配置分组列表
func GetSystemConfigGroups(c *gin.Context) {
	var groups []string

	database.DB.Model(&models.SystemConfig{}).
		Select("DISTINCT group_name").
		Order("group_name ASC").
		Pluck("group_name", &groups)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": groups,
	})
}
