<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <a-row :gutter="[20, 20]">
      <a-col :xs="24" :sm="12" :md="12" :lg="6" v-for="stat in stats" :key="stat.key">
        <a-card :bordered="true" hoverable>
          <a-statistic
            :title="stat.label"
            :value="stat.value"
            :prefix="stat.icon"
            :value-style="{ color: stat.color || '#1890ff', fontSize: '28px', fontWeight: 600 }"
          />
        </a-card>
      </a-col>
    </a-row>

    <!-- 图表区域 -->
    <a-row :gutter="[20, 20]" style="margin-top: 20px">
      <!-- 趋势图表 -->
      <a-col :xs="24" :lg="16">
        <a-card title="最近7天趋势" :bordered="true">
          <div ref="trendChartRef" style="height: 300px"></div>
        </a-card>
      </a-col>

      <!-- 订单统计 -->
      <a-col :xs="24" :lg="8">
        <a-card title="订单统计" :bordered="true">
          <a-statistic
            title="今日充值订单"
            :value="orderStats.todayRecharge"
            style="margin-bottom: 20px"
          />
          <a-statistic
            title="今日提现订单"
            :value="orderStats.todayWithdraw"
            style="margin-bottom: 20px"
          />
          <a-statistic
            title="待支付充值"
            :value="orderStats.pendingRecharge"
            :value-style="{ color: '#faad14' }"
            style="margin-bottom: 20px"
          />
          <a-statistic
            title="待审核提现"
            :value="orderStats.pendingWithdraw"
            :value-style="{ color: '#ff4d4f' }"
          />
        </a-card>
      </a-col>
    </a-row>

    <!-- 游戏统计 -->
    <a-row :gutter="[20, 20]" style="margin-top: 20px">
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="true">
          <a-statistic
            title="总房间数"
            :value="gameStats.totalRooms"
            prefix="🏠"
          />
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="true">
          <a-statistic
            title="进行中房间"
            :value="gameStats.activeRooms"
            prefix="🎮"
            :value-style="{ color: '#52c41a' }"
          />
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="true">
          <a-statistic
            title="今日创建房间"
            :value="gameStats.todayRooms"
            prefix="➕"
          />
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="true">
          <a-statistic
            title="今日完成对局"
            :value="gameStats.todayGameRecords"
            prefix="🎯"
            :value-style="{ color: '#722ed1' }"
          />
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { userAPI } from '../api'

const stats = ref([
  { key: 'users', label: '总用户数', value: 0, icon: '👥', color: '#1890ff' },
  { key: 'todayNewUsers', label: '今日新增', value: 0, icon: '🆕', color: '#52c41a' },
  { key: 'activeUsers', label: '今日活跃', value: 0, icon: '🔥', color: '#faad14' },
  { key: 'balance', label: '总余额', value: 0, icon: '💵', color: '#722ed1' },
  { key: 'todayRecharge', label: '今日充值', value: 0, icon: '⬆️', color: '#52c41a' },
  { key: 'todayWithdraw', label: '今日提现', value: 0, icon: '⬇️', color: '#ff4d4f' },
  { key: 'weekRecharge', label: '本周充值', value: 0, icon: '📈', color: '#1890ff' },
  { key: 'monthRecharge', label: '本月充值', value: 0, icon: '💰', color: '#722ed1' }
])

const orderStats = ref({
  todayRecharge: 0,
  todayWithdraw: 0,
  pendingRecharge: 0,
  pendingWithdraw: 0
})

const gameStats = ref({
  totalRooms: 0,
  activeRooms: 0,
  todayRooms: 0,
  todayGameRecords: 0
})

const trends = ref([])
const trendChartRef = ref(null)
let intervalId = null

const formatNumber = (num) => {
  if (num >= 10000) {
    return (num / 10000).toFixed(1) + '万'
  }
  return num.toString()
}

const formatAmount = (amount) => {
  if (amount >= 10000) {
    return (amount / 10000).toFixed(2) + '万'
  }
  return amount.toFixed(2)
}

const loadStats = async () => {
  try {
    const res = await userAPI.getDashboardStats()
    if (res.code === 200) {
      const data = res.data
      
      // 更新统计卡片
      stats.value[0].value = formatNumber(data.total_users || 0)
      stats.value[1].value = formatNumber(data.today_new_users || 0)
      stats.value[2].value = formatNumber(data.active_users || 0)
      stats.value[3].value = formatAmount(data.total_balance || 0)
      stats.value[4].value = formatAmount(data.today_recharge || 0)
      stats.value[5].value = formatAmount(data.today_withdraw || 0)
      stats.value[6].value = formatAmount(data.week_recharge || 0)
      stats.value[7].value = formatAmount(data.month_recharge || 0)

      // 更新订单统计
      orderStats.value = {
        todayRecharge: data.today_recharge_orders || 0,
        todayWithdraw: data.today_withdraw_orders || 0,
        pendingRecharge: data.pending_recharge || 0,
        pendingWithdraw: data.pending_withdraw || 0
      }

      // 更新游戏统计
      gameStats.value = {
        totalRooms: data.total_rooms || 0,
        activeRooms: data.active_rooms || 0,
        todayRooms: data.today_rooms || 0,
        todayGameRecords: data.today_game_records || 0
      }
    }
  } catch (error) {
    console.error('加载统计失败:', error)
  }
}

const loadTrends = async () => {
  try {
    const res = await userAPI.getDashboardTrends()
    if (res.code === 200) {
      trends.value = res.data || []
      renderChart()
    }
  } catch (error) {
    console.error('加载趋势数据失败:', error)
  }
}

const renderChart = () => {
  if (!trendChartRef.value || trends.value.length === 0) return

  // 使用简单的Canvas绘制图表
  const canvas = document.createElement('canvas')
  canvas.width = trendChartRef.value.clientWidth - 40
  canvas.height = 280
  trendChartRef.value.innerHTML = ''
  trendChartRef.value.appendChild(canvas)

  const ctx = canvas.getContext('2d')
  const padding = 40
  const chartWidth = canvas.width - padding * 2
  const chartHeight = canvas.height - padding * 2

  // 找到最大值用于缩放
  const maxRecharge = Math.max(...trends.value.map(t => t.recharge || 0))
  const maxWithdraw = Math.max(...trends.value.map(t => t.withdraw || 0))
  const maxUsers = Math.max(...trends.value.map(t => t.new_users || 0))
  const maxRecords = Math.max(...trends.value.map(t => t.game_records || 0))
  const maxValue = Math.max(maxRecharge, maxWithdraw, maxUsers * 10, maxRecords) || 1

  // 绘制网格线
  ctx.strokeStyle = '#f0f0f0'
  ctx.lineWidth = 1
  for (let i = 0; i <= 5; i++) {
    const y = padding + (chartHeight / 5) * i
    ctx.beginPath()
    ctx.moveTo(padding, y)
    ctx.lineTo(padding + chartWidth, y)
    ctx.stroke()
  }

  // 绘制数据线
  const step = chartWidth / (trends.value.length - 1)
  
  // 充值线（绿色）
  ctx.strokeStyle = '#52c41a'
  ctx.lineWidth = 2
  ctx.beginPath()
  trends.value.forEach((item, index) => {
    const x = padding + step * index
    const y = padding + chartHeight - (item.recharge / maxValue) * chartHeight
    if (index === 0) {
      ctx.moveTo(x, y)
    } else {
      ctx.lineTo(x, y)
    }
  })
  ctx.stroke()

  // 提现线（红色）
  ctx.strokeStyle = '#ff4d4f'
  ctx.lineWidth = 2
  ctx.beginPath()
  trends.value.forEach((item, index) => {
    const x = padding + step * index
    const y = padding + chartHeight - (item.withdraw / maxValue) * chartHeight
    if (index === 0) {
      ctx.moveTo(x, y)
    } else {
      ctx.lineTo(x, y)
    }
  })
  ctx.stroke()

  // 新增用户线（蓝色）
  ctx.strokeStyle = '#1890ff'
  ctx.lineWidth = 2
  ctx.beginPath()
  trends.value.forEach((item, index) => {
    const x = padding + step * index
    const y = padding + chartHeight - ((item.new_users * 10) / maxValue) * chartHeight
    if (index === 0) {
      ctx.moveTo(x, y)
    } else {
      ctx.lineTo(x, y)
    }
  })
  ctx.stroke()

  // 绘制日期标签
  ctx.fillStyle = '#666'
  ctx.font = '12px Arial'
  ctx.textAlign = 'center'
  trends.value.forEach((item, index) => {
    const x = padding + step * index
    ctx.fillText(item.date, x, canvas.height - 10)
  })

  // 绘制图例
  const legendY = 20
  ctx.font = '12px Arial'
  ctx.textAlign = 'left'
  
  ctx.strokeStyle = '#52c41a'
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.moveTo(padding, legendY)
  ctx.lineTo(padding + 20, legendY)
  ctx.stroke()
  ctx.fillStyle = '#000'
  ctx.fillText('充值', padding + 25, legendY + 4)

  ctx.strokeStyle = '#ff4d4f'
  ctx.beginPath()
  ctx.moveTo(padding + 60, legendY)
  ctx.lineTo(padding + 80, legendY)
  ctx.stroke()
  ctx.fillText('提现', padding + 85, legendY + 4)

  ctx.strokeStyle = '#1890ff'
  ctx.beginPath()
  ctx.moveTo(padding + 120, legendY)
  ctx.lineTo(padding + 140, legendY)
  ctx.stroke()
  ctx.fillText('新增用户(×10)', padding + 145, legendY + 4)
}

onMounted(() => {
  loadStats()
  loadTrends()
  
  // 每30秒刷新一次数据
  intervalId = setInterval(() => {
    loadStats()
    loadTrends()
  }, 30000)
})

onUnmounted(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})
</script>

<style scoped>
.dashboard {
  padding: 0;
}

.dashboard :deep(.ant-card) {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.dashboard :deep(.ant-statistic-title) {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.dashboard :deep(.ant-statistic-content) {
  font-size: 24px;
}
</style>
