package services

import (
	"fmt"
	"time"

	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
)

// SendOrderNotification 发送订单状态通知
func SendOrderNotification(userID uint, orderType string, orderID string, status string, amount float64, message string) {
	now := time.Now().Unix()

	var title string
	var msgType string

	switch orderType {
	case "recharge":
		switch status {
		case "paid":
			title = "充值成功"
			msgType = "success"
		case "failed":
			title = "充值失败"
			msgType = "error"
		case "expired":
			title = "充值订单已过期"
			msgType = "warning"
		default:
			title = "充值订单状态更新"
			msgType = "info"
		}
	case "withdraw":
		switch status {
		case "approved":
			title = "提现审核通过"
			msgType = "success"
		case "rejected":
			title = "提现审核拒绝"
			msgType = "error"
		case "completed":
			title = "提现已完成"
			msgType = "success"
		default:
			title = "提现订单状态更新"
			msgType = "info"
		}
	default:
		title = "订单状态更新"
		msgType = "info"
	}

	// 构建消息内容
	content := message
	if content == "" {
		switch orderType {
		case "recharge":
			if status == "paid" {
				content = fmt.Sprintf("您的充值订单 %s 已成功支付，充值金额 %.2f USDT 已到账。", orderID, amount)
			} else if status == "expired" {
				content = fmt.Sprintf("您的充值订单 %s 已过期，请重新创建订单。", orderID)
			}
		case "withdraw":
			if status == "approved" {
				content = fmt.Sprintf("您的提现订单 %s 审核已通过，金额 %.2f USDT 将在24小时内到账。", orderID, amount)
			} else if status == "rejected" {
				content = fmt.Sprintf("您的提现订单 %s 审核未通过，如有疑问请联系客服。", orderID)
			}
		}
	}

	// 创建用户消息
	userMessage := models.UserMessage{
		UserID:    userID,
		Type:      msgType,
		Title:     title,
		Content:   content,
		RelatedID: orderID,
		IsRead:    false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 异步保存消息
	go func() {
		database.DB.Create(&userMessage)
	}()
}
