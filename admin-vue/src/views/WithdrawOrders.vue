<template>
  <div class="withdraw-orders">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">提现订单</span>
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
            <a-select-option value="1">待审核</a-select-option>
            <a-select-option value="2">已通过</a-select-option>
            <a-select-option value="3">已拒绝</a-select-option>
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
          <template v-else-if="column.key === 'to_address'">
            <a-typography-text :ellipsis="{ tooltip: true }" style="max-width: 300px">
              {{ record.to_address }}
            </a-typography-text>
          </template>
          <template v-else-if="column.key === 'tx_hash'">
            <a-typography-text v-if="record.tx_hash" :ellipsis="{ tooltip: true }" style="max-width: 300px">
              {{ record.tx_hash }}
            </a-typography-text>
            <span v-else style="color: #999">-</span>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space v-if="record.status === 1">
              <a-button type="primary" size="small" @click="handleAudit(record, true)">通过</a-button>
              <a-button type="primary" danger size="small" @click="handleAudit(record, false)">拒绝</a-button>
            </a-space>
            <span v-else style="color: #999">-</span>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, h } from 'vue'
import { message, Modal, Input } from 'ant-design-vue'
import { withdrawAPI } from '../api'

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
    title: '提现地址',
    dataIndex: 'to_address',
    key: 'to_address',
    width: 300
  },
  {
    title: '交易哈希',
    dataIndex: 'tx_hash',
    key: 'tx_hash',
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

const getStatusColor = (status) => {
  const colors = { 1: 'orange', 2: 'green', 3: 'red' }
  return colors[status] || 'default'
}

const getStatusText = (status) => {
  const texts = { 1: '待审核', 2: '已通过', 3: '已拒绝' }
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
    
    const res = await withdrawAPI.getWithdrawOrders(params)
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

const handleAudit = async (row, approve) => {
  try {
    let remark = ''
    
    if (approve) {
      Modal.confirm({
        title: '审核提现订单',
        content: '确认通过该提现订单？',
        onOk: async () => {
          await doAudit(row, approve, remark)
        }
      })
    } else {
      const remarkRef = ref('')
      Modal.confirm({
        title: '审核提现订单',
        content: () => h('div', [
          h('p', '请输入拒绝原因：'),
          h(Input.TextArea, {
            value: remarkRef.value,
            'onUpdate:value': (val) => {
              remarkRef.value = val
            },
            placeholder: '请输入拒绝原因',
            rows: 4,
            style: { marginTop: '8px' }
          })
        ]),
        onOk: async () => {
          remark = remarkRef.value || ''
          await doAudit(row, approve, remark)
        }
      })
    }
  } catch (error) {
    console.error('审核失败:', error)
  }
}

const doAudit = async (row, approve, remark) => {
  try {
    const res = await withdrawAPI.auditWithdrawOrder(row.order_id, {
      approve,
      remark
    })
    
    if (res.code === 200) {
      message.success(approve ? '审核通过' : '已拒绝')
      loadOrders()
    } else {
      message.error(res.message || '审核失败')
    }
  } catch (error) {
    message.error('审核失败：' + (error.response?.data?.message || error.message))
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
.withdraw-orders {
  padding: 0;
}
</style>
