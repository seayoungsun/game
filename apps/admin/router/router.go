package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/apps/admin/handlers"
	"github.com/kaifa/game-platform/apps/admin/middleware"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/pkg/utils"
)

// Setup 装配所有路由
func Setup(cfg *config.Config) *gin.Engine {
	r := gin.New()

	// 通用中间件
	r.Use(middleware.CORSMiddleware()) // CORS跨域支持
	r.Use(ginLogger())
	r.Use(ginRecovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "admin-server",
			"port":   8082,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API路由组
	v1 := r.Group("/api/v1")

	// 认证相关（不需要Token）
	auth := v1.Group("/auth")
	{
		auth.POST("/login", handlers.Login)
		auth.POST("/logout", middleware.AdminAuthMiddleware(), handlers.Logout)
	}

	// 需要认证的路由
	admin := v1.Group("/admin")
	admin.Use(middleware.AdminAuthMiddleware())
	admin.Use(middleware.OperationLogMiddleware()) // 操作日志中间件
	{
		// 管理员信息
		admin.GET("/profile", handlers.GetProfile)
		admin.GET("/permissions", handlers.GetPermissions)

		// 仪表盘
		dashboard := admin.Group("/dashboard")
		dashboard.Use(middleware.RequirePermission(utils.PermissionDashboardView))
		{
			dashboard.GET("/stats", handlers.GetDashboardStats)
			dashboard.GET("/trends", handlers.GetDashboardTrends)
		}

		// 用户管理
		users := admin.Group("/users")
		users.Use(middleware.RequirePermission(utils.PermissionUsersList))
		{
			users.GET("", handlers.GetUsers)
			users.GET("/:id", middleware.RequirePermission(utils.PermissionUsersDetail), handlers.GetUserDetail)
			users.PUT("/:id", middleware.RequirePermission(utils.PermissionUsersUpdate), handlers.UpdateUser)
		}

		// 充值订单
		rechargeOrders := admin.Group("/recharge-orders")
		rechargeOrders.Use(middleware.RequirePermission(utils.PermissionRechargeOrdersList))
		{
			rechargeOrders.GET("", handlers.GetRechargeOrders)
		}

		// 提现订单
		withdrawOrders := admin.Group("/withdraw-orders")
		withdrawOrders.Use(middleware.RequirePermission(utils.PermissionWithdrawOrdersList))
		{
			withdrawOrders.GET("", handlers.GetWithdrawOrders)
			withdrawOrders.POST("/:orderId/audit", middleware.RequirePermission(utils.PermissionWithdrawOrdersAudit), handlers.AuditWithdrawOrder)
		}

		// 充值地址
		depositAddresses := admin.Group("/deposit-addresses")
		depositAddresses.Use(middleware.RequirePermission(utils.PermissionDepositAddressesList))
		{
			depositAddresses.GET("", handlers.GetDepositAddresses)
		}

		// USDT归集
		payments := admin.Group("/payments")
		{
			payments.POST("/collect", middleware.RequirePermission(utils.PermissionPaymentsCollect), handlers.CollectUSDT)
			payments.POST("/batch-collect", middleware.RequirePermission(utils.PermissionPaymentsBatchCollect), handlers.BatchCollectUSDT)
		}

		// 系统管理 - 角色管理
		roles := admin.Group("/roles")
		roles.Use(middleware.RequirePermission(utils.PermissionRolesList))
		{
			roles.GET("", handlers.GetRoles)
			roles.GET("/:id", handlers.GetRole)
			roles.POST("", middleware.RequirePermission(utils.PermissionRolesCreate), handlers.CreateRole)
			roles.PUT("/:id", middleware.RequirePermission(utils.PermissionRolesUpdate), handlers.UpdateRole)
			roles.DELETE("/:id", middleware.RequirePermission(utils.PermissionRolesDelete), handlers.DeleteRole)
		}

		// 系统管理 - 权限管理
		permissions := admin.Group("/permissions")
		{
			permissions.GET("/all", handlers.GetAllPermissions)
		}

		// 系统管理 - 管理员管理
		admins := admin.Group("/admins")
		admins.Use(middleware.RequirePermission(utils.PermissionAdminsList))
		{
			admins.GET("", handlers.GetAdmins)
			admins.GET("/:id", handlers.GetAdmin)
			admins.POST("", middleware.RequirePermission(utils.PermissionAdminsCreate), handlers.CreateAdmin)
			admins.PUT("/:id", middleware.RequirePermission(utils.PermissionAdminsUpdate), handlers.UpdateAdmin)
			admins.DELETE("/:id", middleware.RequirePermission(utils.PermissionAdminsDelete), handlers.DeleteAdmin)
		}

		// 操作日志
		logs := admin.Group("/operation-logs")
		logs.Use(middleware.RequirePermission(utils.PermissionRolesList)) // 使用已有权限，后续可以添加专门权限
		{
			logs.GET("", handlers.GetOperationLogs)
			logs.GET("/:id", handlers.GetOperationLog)
			logs.DELETE("/:id", handlers.DeleteOperationLog)
			logs.POST("/batch-delete", handlers.BatchDeleteOperationLogs)
			logs.POST("/clean", handlers.CleanOldLogs)
		}

		// 系统设置
		configs := admin.Group("/system-configs")
		{
			configs.GET("", handlers.GetSystemConfigs)
			configs.GET("/groups", handlers.GetSystemConfigGroups)
			configs.GET("/:key", handlers.GetSystemConfig)
			configs.PUT("/:key", handlers.UpdateSystemConfig)
			configs.POST("", handlers.CreateSystemConfig)
			configs.DELETE("/:key", handlers.DeleteSystemConfig)
		}

		// 消息管理
		messages := admin.Group("/messages")
		{
			// 公告管理
			announcements := messages.Group("/announcements")
			{
				announcements.GET("", handlers.GetAnnouncements)
				announcements.GET("/:id", handlers.GetAnnouncement)
				announcements.POST("", handlers.CreateAnnouncement)
				announcements.PUT("/:id", handlers.UpdateAnnouncement)
				announcements.DELETE("/:id", handlers.DeleteAnnouncement)
			}

			// 用户消息管理
			userMessages := messages.Group("/user-messages")
			{
				userMessages.GET("", handlers.GetUserMessages)
				userMessages.POST("/send", handlers.SendUserMessage)
				userMessages.DELETE("/:id", handlers.DeleteUserMessage)
				userMessages.POST("/batch-delete", handlers.BatchDeleteUserMessages)
			}
		}
	}

	return r
}

// 日志中间件
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		_ = time.Since(start)
		_ = path
	}
}

// 恢复中间件
func ginRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}
