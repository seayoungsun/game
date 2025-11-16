<template>
  <div class="recharge-orders">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">充值订单</span>
      </template>
      <template #extra>
        <a-space>
          <a-select
            v-model:value="filters.status"
            placeholder="状态"
            allow-clear
            style="width: 120px"
            @change="loadOrders"
          >
            <a-select-option value="1">待支付</a-select-option>
            <a-select-option value="2">已支付</a-select-option>
            <a-select-option value="3">已取消</a-select-option>
          </a-select>
          <a-select
            v-model:value="filters.chain_type"
            placeholder="链类型"
            allow-clear
            style="width: 120px"
            @change="loadOrders"
          >
            <a-select-option value="trc20">TRC20</a-select-option>
            <a-select-option value="erc20">ERC20</a-select-option>
          </a-select>
          <a-button type="primary" @click="loadOrders">刷新</a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="orders"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.order_id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'amount'">
            {{ parseFloat(record.amount || 0).toFixed(2) }}
          </template>
          <template v-else-if="column.key === 'chain_type'">
            <a-tag>{{ (record.chain_type || '').toUpperCase() }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'deposit_address'">
            <a-typography-text :ellipsis="{ tooltip: true }" style="max-width: 300px">
              {{ record.deposit_address }}
            </a-typography-text>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { rechargeAPI } from '../api'

const orders = ref([])
const loading = ref(false)
const filters = ref({
  status: undefined,
  chain_type: undefined
})
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const columns = [
  {
    title: '订单号',
    dataIndex: 'order_id',
    key: 'order_id',
    width: 200
  },
  {
    title: '用户ID',
    dataIndex: 'user_id',
    key: 'user_id',
    width: 100
  },
  {
    title: '金额',
    dataIndex: 'amount',
    key: 'amount',
    width: 120
  },
  {
    title: '链类型',
    dataIndex: 'chain_type',
    key: 'chain_type',
    width: 100
  },
  {
    title: '充值地址',
    dataIndex: 'deposit_address',
    key: 'deposit_address',
    width: 300
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: 180
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

const getStatusColor = (status) => {
  const colors = { 1: 'orange', 2: 'green', 3: 'default' }
  return colors[status] || 'default'
}

const getStatusText = (status) => {
  const texts = { 1: '待支付', 2: '已支付', 3: '已取消' }
  return texts[status] || '-'
}

const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const loadOrders = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.value.status !== undefined) params.status = filters.value.status
    if (filters.value.chain_type !== undefined) params.chain_type = filters.value.chain_type
    
    const res = await rechargeAPI.getRechargeOrders(params)
    if (res.code === 200) {
      orders.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载订单列表失败')
    }
  } catch (error) {
    message.error('加载订单列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadOrders()
}

onMounted(() => {
  loadOrders()
})
</script>

<style scoped>
.recharge-orders {
  padding: 0;
}
</style>
