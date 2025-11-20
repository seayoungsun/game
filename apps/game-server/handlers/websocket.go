package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/utils"
	"go.uber.org/zap"
)

var (
	upgraderInstance *websocket.Upgrader
	hubInstance      HubInterface
	newClientFunc    NewClientFunc
)

// HubInterface Hub 接口
type HubInterface interface {
	RegisterClient(client ClientInterface) bool
	GetUserClient(userID uint) ClientInterface
	GetRoomClients(roomID string) []ClientInterface
	BroadcastMessage(msg MessageInterface)
	PublishSystemMessage(msgType, roomID string, data map[string]interface{}) error
}

// ClientInterface Client 接口
type ClientInterface interface {
	Start()
	SendMessage(msg MessageInterface)
	GetUserID() uint
}

// MessageInterface Message 接口
type MessageInterface interface {
	GetType() string
	GetRoomID() string
	GetUserID() uint
	GetRawData() interface{}
}

// NewClientFunc 创建客户端的函数类型
type NewClientFunc func(conn *websocket.Conn, ip string, userID uint) ClientInterface

// NewMessageFunc 创建消息的函数类型
type NewMessageFunc func(msgType, roomID string, userID uint, rawData interface{}) MessageInterface

var newMessageFunc NewMessageFunc

// SetUpgrader 设置 WebSocket Upgrader
func SetUpgrader(u *websocket.Upgrader) {
	upgraderInstance = u
}

// SetHub 设置 Hub 实例
func SetHub(h HubInterface) {
	hubInstance = h
}

// SetNewClientFunc 设置创建客户端的函数
func SetNewClientFunc(f NewClientFunc) {
	newClientFunc = f
}

// SetNewMessageFunc 设置创建消息的函数
func SetNewMessageFunc(f NewMessageFunc) {
	newMessageFunc = f
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
	// 获取Token（从query参数或header）
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(401, gin.H{"code": 401, "message": "缺少认证token"})
		return
	}

	// 验证Token
	claims, err := utils.ParseToken(token)
	if err != nil {
		logger.Logger.Warn("Token验证失败", zap.Error(err))
		c.JSON(401, gin.H{"code": 401, "message": "无效的token"})
		return
	}

	// 升级到WebSocket连接
	conn, err := upgraderInstance.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if !c.Writer.Written() {
			c.JSON(500, gin.H{
				"code":    500,
				"message": "WebSocket升级失败",
			})
		}
		logger.Logger.Error("WebSocket升级失败",
			zap.Error(err),
			zap.Uint("user_id", claims.UserID),
			zap.String("ip", c.ClientIP()),
			zap.String("remote_addr", c.Request.RemoteAddr),
		)
		return
	}

	logger.Logger.Info("新的WebSocket连接",
		zap.Uint("user_id", claims.UserID),
		zap.String("ip", c.ClientIP()),
	)

	// 创建客户端
	client := newClientFunc(conn, c.ClientIP(), claims.UserID)

	// 注册到Hub
	if !hubInstance.RegisterClient(client) {
		logger.Logger.Error("Hub注册channel已满，无法注册客户端",
			zap.Uint("user_id", claims.UserID),
			zap.String("ip", c.ClientIP()),
		)
		conn.Close()
		return
	}

	// 启动读写goroutine
	client.Start()

	// 发送连接成功消息
	client.SendMessage(newMessageFunc("connected", "", claims.UserID, map[string]interface{}{
		"message": "连接成功",
		"user_id": claims.UserID,
	}))
}
