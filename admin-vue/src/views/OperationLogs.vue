<template>
  <div class="operation-logs">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">操作日志</span>
      </template>
      <template #extra>
        <a-space>
          <RangePicker
            v-model:value="dateRange"
            format="YYYY-MM-DD"
            @change="handleDateChange"
            style="width: 240px"
          />
          <a-select
            v-model:value="filters.module"
            placeholder="选择模块"
            style="width: 150px"
            allow-clear
            @change="handleSearch"
          >
            <a-select-option value="users">用户管理</a-select-option>
            <a-select-option value="roles">角色管理</a-select-option>
            <a-select-option value="admins">管理员管理</a-select-option>
            <a-select-option value="recharge-orders">充值订单</a-select-option>
            <a-select-option value="withdraw-orders">提现订单</a-select-option>
            <a-select-option value="dashboard">仪表盘</a-select-option>
          </a-select>
          <a-select
            v-model:value="filters.status"
            placeholder="选择状态"
            style="width: 120px"
            allow-clear
            @change="handleSearch"
          >
            <a-select-option :value="1">成功</a-select-option>
            <a-select-option :value="2">失败</a-select-option>
          </a-select>
          <a-button type="primary" @click="handleSearch">搜索</a-button>
          <a-button @click="handleClean">清理旧日志</a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="logs"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'error'">
              {{ record.status === 1 ? '成功' : '失败' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'module'">
            <a-tag color="blue">{{ getModuleName(record.module) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'duration'">
            {{ record.duration }}ms
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="handleViewDetail(record)">查看</a-button>
              <a-button type="link" size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 日志详情对话框 -->
    <a-modal
      v-model:open="detailVisible"
      title="操作日志详情"
      :footer="null"
      width="800px"
    >
      <a-descriptions :column="2" bordered v-if="currentLog">
        <a-descriptions-item label="ID">{{ currentLog.id }}</a-descriptions-item>
        <a-descriptions-item label="管理员">{{ currentLog.admin_name }}</a-descriptions-item>
        <a-descriptions-item label="模块">{{ getModuleName(currentLog.module) }}</a-descriptions-item>
        <a-descriptions-item label="动作">{{ currentLog.action }}</a-descriptions-item>
        <a-descriptions-item label="请求方法">{{ currentLog.method }}</a-descriptions-item>
        <a-descriptions-item label="请求路径">{{ currentLog.path }}</a-descriptions-item>
        <a-descriptions-item label="IP地址">{{ currentLog.ip }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="currentLog.status === 1 ? 'success' : 'error'">
            {{ currentLog.status === 1 ? '成功' : '失败' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="耗时">{{ currentLog.duration }}ms</a-descriptions-item>
        <a-descriptions-item label="操作时间">{{ formatTime(currentLog.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="请求参数" :span="2">
          <pre style="background: #f5f5f5; padding: 10px; border-radius: 4px; max-height: 200px; overflow: auto;">{{ formatJSON(currentLog.request) }}</pre>
        </a-descriptions-item>
        <a-descriptions-item label="响应结果" :span="2">
          <pre style="background: #f5f5f5; padding: 10px; border-radius: 4px; max-height: 200px; overflow: auto;">{{ formatJSON(currentLog.response) }}</pre>
        </a-descriptions-item>
        <a-descriptions-item label="错误信息" :span="2" v-if="currentLog.error_msg">
          <pre style="background: #fff1f0; padding: 10px; border-radius: 4px; color: #ff4d4f;">{{ currentLog.error_msg }}</pre>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>

    <!-- 清理旧日志对话框 -->
    <a-modal
      v-model:open="cleanVisible"
      title="清理旧日志"
      @ok="handleConfirmClean"
      :confirm-loading="cleaning"
    >
      <a-form layout="vertical">
        <a-form-item label="保留最近天数">
          <a-input-number v-model:value="cleanDays" :min="1" :max="365" style="width: 100%" />
          <div style="margin-top: 8px; color: #999;">
            将删除 {{ cleanDays }} 天前的日志记录
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal, DatePicker } from 'ant-design-vue'
import { operationLogAPI } from '../api'

const RangePicker = DatePicker.RangePicker

const logs = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const detailVisible = ref(false)
const cleanVisible = ref(false)
const cleaning = ref(false)
const currentLog = ref(null)
const cleanDays = ref(30)
const dateRange = ref(null)

const filters = ref({
  module: undefined,
  status: undefined,
  date_start: '',
  date_end: ''
})

const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80
  },
  {
    title: '管理员',
    dataIndex: 'admin_name',
    key: 'admin_name',
    width: 120
  },
  {
    title: '模块',
    dataIndex: 'module',
    key: 'module',
    width: 120
  },
  {
    title: '动作',
    dataIndex: 'action',
    key: 'action',
    width: 100
  },
  {
    title: '请求方法',
    dataIndex: 'method',
    key: 'method',
    width: 100
  },
  {
    title: '请求路径',
    dataIndex: 'path',
    key: 'path',
    width: 200
  },
  {
    title: 'IP地址',
    dataIndex: 'ip',
    key: 'ip',
    width: 150
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100
  },
  {
    title: '耗时',
    dataIndex: 'duration',
    key: 'duration',
    width: 100
  },
  {
    title: '操作时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right'
  }
]

const paginationConfig = computed(() => ({
  current: currentPage.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total) => `共 ${total} 条`,
  pageSizeOptions: ['10', '20', '50', '100']
}))

const getModuleName = (module) => {
  const moduleMap = {
    'users': '用户管理',
    'roles': '角色管理',
    'admins': '管理员管理',
    'recharge-orders': '充值订单',
    'withdraw-orders': '提现订单',
    'deposit-addresses': '充值地址',
    'payments': '支付管理',
    'dashboard': '仪表盘',
    'permissions': '权限管理',
    'operation-logs': '操作日志',
    'system-configs': '系统设置'
  }
  return moduleMap[module] || module
}

const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const formatJSON = (str) => {
  if (!str) return '-'
  try {
    const obj = JSON.parse(str)
    return JSON.stringify(obj, null, 2)
  } catch {
    return str
  }
}

const loadLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.value.module) {
      params.module = filters.value.module
    }
    if (filters.value.status) {
      params.status = filters.value.status
    }
    if (filters.value.date_start) {
      params.date_start = filters.value.date_start
    }
    if (filters.value.date_end) {
      params.date_end = filters.value.date_end
    }
    
    const res = await operationLogAPI.getOperationLogs(params)
    if (res.code === 200) {
      logs.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载日志列表失败')
    }
  } catch (error) {
    message.error('加载日志列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadLogs()
}

const handleDateChange = (dates) => {
  if (dates && dates.length === 2) {
    // Ant Design Vue 4.x 的日期选择器返回的是 dayjs 对象数组
    // 如果没有 dayjs，则使用 Date 对象
    const startDate = dates[0]
    const endDate = dates[1]
    
    // 检查是否是 dayjs 对象
    if (startDate && typeof startDate.startOf === 'function') {
      filters.value.date_start = Math.floor(startDate.startOf('day').valueOf() / 1000).toString()
      filters.value.date_end = Math.floor(endDate.endOf('day').valueOf() / 1000).toString()
    } else {
      // 使用 Date 对象
      const start = new Date(startDate)
      start.setHours(0, 0, 0, 0)
      const end = new Date(endDate)
      end.setHours(23, 59, 59, 999)
      filters.value.date_start = Math.floor(start.getTime() / 1000).toString()
      filters.value.date_end = Math.floor(end.getTime() / 1000).toString()
    }
  } else {
    filters.value.date_start = ''
    filters.value.date_end = ''
  }
  handleSearch()
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadLogs()
}

const handleViewDetail = (record) => {
  currentLog.value = record
  detailVisible.value = true
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除这条日志吗？`,
    onOk: async () => {
      try {
        const res = await operationLogAPI.deleteOperationLog(record.id)
        if (res.code === 200) {
          message.success('删除成功')
          loadLogs()
        } else {
          message.error(res.message || '删除失败')
        }
      } catch (error) {
        message.error('删除失败：' + (error.response?.data?.message || error.message))
      }
    }
  })
}

const handleClean = () => {
  cleanVisible.value = true
}

const handleConfirmClean = async () => {
  cleaning.value = true
  try {
    const res = await operationLogAPI.cleanOldLogs(cleanDays.value)
    if (res.code === 200) {
      message.success(`清理成功，删除了 ${res.data.deleted_count || 0} 条日志`)
      cleanVisible.value = false
      loadLogs()
    } else {
      message.error(res.message || '清理失败')
    }
  } catch (error) {
    message.error('清理失败：' + (error.response?.data?.message || error.message))
  } finally {
    cleaning.value = false
  }
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.operation-logs {
  padding: 0;
}
</style>

