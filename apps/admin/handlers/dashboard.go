package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
)

// GetDashboardStats 获取仪表盘统计数据
func GetDashboardStats(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	todayEnd := todayStart + 86400 - 1

	// 本周开始（周一）
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日算作第7天
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location()).Unix()
	weekEnd := now.Unix()

	// 本月开始
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	monthEnd := now.Unix()

	var stats struct {
		// 用户统计
		TotalUsers    int64 `json:"total_users"`
		TodayNewUsers int64 `json:"today_new_users"`
		WeekNewUsers  int64 `json:"week_new_users"`
		MonthNewUsers int64 `json:"month_new_users"`
		ActiveUsers   int64 `json:"active_users"` // 今日活跃用户（今日有登录的用户）

		// 余额统计
		TotalBalance  float64 `json:"total_balance"`
		TodayRecharge float64 `json:"today_recharge"`
		WeekRecharge  float64 `json:"week_recharge"`
		MonthRecharge float64 `json:"month_recharge"`
		TodayWithdraw float64 `json:"today_withdraw"`
		WeekWithdraw  float64 `json:"week_withdraw"`
		MonthWithdraw float64 `json:"month_withdraw"`

		// 订单统计
		TodayRechargeOrders int64 `json:"today_recharge_orders"`
		TodayWithdrawOrders int64 `json:"today_withdraw_orders"`
		PendingRecharge     int64 `json:"pending_recharge"` // 待支付充值订单
		PendingWithdraw     int64 `json:"pending_withdraw"` // 待审核提现订单

		// 游戏统计
		TotalRooms       int64 `json:"total_rooms"`
		ActiveRooms      int64 `json:"active_rooms"`       // 进行中的房间
		TodayRooms       int64 `json:"today_rooms"`        // 今日创建的房间
		TodayGameRecords int64 `json:"today_game_records"` // 今日完成的游戏对局
	}

	// 总用户数
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// 今日新增用户
	database.DB.Model(&models.User{}).Where("created_at >= ? AND created_at <= ?", todayStart, todayEnd).Count(&stats.TodayNewUsers)

	// 本周新增用户
	database.DB.Model(&models.User{}).Where("created_at >= ? AND created_at <= ?", weekStart, weekEnd).Count(&stats.WeekNewUsers)

	// 本月新增用户
	database.DB.Model(&models.User{}).Where("created_at >= ? AND created_at <= ?", monthStart, monthEnd).Count(&stats.MonthNewUsers)

	// 今日活跃用户（今日有登录记录的用户）
	database.DB.Table("user_logins").
		Where("created_at >= ? AND created_at <= ?", todayStart, todayEnd).
		Distinct("user_id").
		Count(&stats.ActiveUsers)

	// 总余额
	database.DB.Model(&models.User{}).Select("COALESCE(SUM(balance), 0)").Scan(&stats.TotalBalance)

	// 今日充值金额
	database.DB.Model(&models.RechargeOrder{}).
		Where("status = 2 AND paid_at >= ? AND paid_at <= ?", todayStart, todayEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TodayRecharge)

	// 本周充值金额
	database.DB.Model(&models.RechargeOrder{}).
		Where("status = 2 AND paid_at >= ? AND paid_at <= ?", weekStart, weekEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.WeekRecharge)

	// 本月充值金额
	database.DB.Model(&models.RechargeOrder{}).
		Where("status = 2 AND paid_at >= ? AND paid_at <= ?", monthStart, monthEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.MonthRecharge)

	// 今日提现金额
	database.DB.Model(&models.WithdrawOrder{}).
		Where("status = 2 AND audit_at >= ? AND audit_at <= ?", todayStart, todayEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.TodayWithdraw)

	// 本周提现金额
	database.DB.Model(&models.WithdrawOrder{}).
		Where("status = 2 AND audit_at >= ? AND audit_at <= ?", weekStart, weekEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.WeekWithdraw)

	// 本月提现金额
	database.DB.Model(&models.WithdrawOrder{}).
		Where("status = 2 AND audit_at >= ? AND audit_at <= ?", monthStart, monthEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&stats.MonthWithdraw)

	// 今日充值订单数
	database.DB.Model(&models.RechargeOrder{}).
		Where("created_at >= ? AND created_at <= ?", todayStart, todayEnd).
		Count(&stats.TodayRechargeOrders)

	// 今日提现订单数
	database.DB.Model(&models.WithdrawOrder{}).
		Where("created_at >= ? AND created_at <= ?", todayStart, todayEnd).
		Count(&stats.TodayWithdrawOrders)

	// 待支付充值订单
	database.DB.Model(&models.RechargeOrder{}).Where("status = 1").Count(&stats.PendingRecharge)

	// 待审核提现订单
	database.DB.Model(&models.WithdrawOrder{}).Where("status = 1").Count(&stats.PendingWithdraw)

	// 总房间数
	database.DB.Model(&models.GameRoom{}).Count(&stats.TotalRooms)

	// 进行中的房间
	database.DB.Model(&models.GameRoom{}).Where("status = 2").Count(&stats.ActiveRooms)

	// 今日创建的房间
	database.DB.Model(&models.GameRoom{}).
		Where("created_at >= ? AND created_at <= ?", todayStart, todayEnd).
		Count(&stats.TodayRooms)

	// 今日完成的游戏对局
	database.DB.Model(&models.GameRecord{}).
		Where("end_time >= ? AND end_time <= ?", todayStart, todayEnd).
		Count(&stats.TodayGameRecords)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": stats,
	})
}

// GetDashboardTrends 获取仪表盘趋势数据（最近7天）
func GetDashboardTrends(c *gin.Context) {
	now := time.Now()
	var trends []struct {
		Date        string  `json:"date"`
		NewUsers    int64   `json:"new_users"`
		Recharge    float64 `json:"recharge"`
		Withdraw    float64 `json:"withdraw"`
		GameRecords int64   `json:"game_records"`
	}

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Unix()
		dateEnd := dateStart + 86400 - 1

		trend := struct {
			Date        string  `json:"date"`
			NewUsers    int64   `json:"new_users"`
			Recharge    float64 `json:"recharge"`
			Withdraw    float64 `json:"withdraw"`
			GameRecords int64   `json:"game_records"`
		}{
			Date: date.Format("01-02"),
		}

		// 当日新增用户
		database.DB.Model(&models.User{}).
			Where("created_at >= ? AND created_at <= ?", dateStart, dateEnd).
			Count(&trend.NewUsers)

		// 当日充值
		database.DB.Model(&models.RechargeOrder{}).
			Where("status = 2 AND paid_at >= ? AND paid_at <= ?", dateStart, dateEnd).
			Select("COALESCE(SUM(amount), 0)").Scan(&trend.Recharge)

		// 当日提现
		database.DB.Model(&models.WithdrawOrder{}).
			Where("status = 2 AND audit_at >= ? AND audit_at <= ?", dateStart, dateEnd).
			Select("COALESCE(SUM(amount), 0)").Scan(&trend.Withdraw)

		// 当日完成的游戏对局
		database.DB.Model(&models.GameRecord{}).
			Where("end_time >= ? AND end_time <= ?", dateStart, dateEnd).
			Count(&trend.GameRecords)

		trends = append(trends, trend)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": trends,
	})
}
