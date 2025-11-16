<template>
  <div class="deposit-addresses">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">充值地址</span>
      </template>
      <template #extra>
        <a-space>
          <a-select
            v-model:value="filters.chain_type"
            placeholder="链类型"
            allow-clear
            style="width: 120px"
            @change="loadAddresses"
          >
            <a-select-option value="trc20">TRC20</a-select-option>
            <a-select-option value="erc20">ERC20</a-select-option>
          </a-select>
          <a-button type="primary" @click="loadAddresses">刷新</a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="addresses"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.user_id + record.chain_type"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'chain_type'">
            <a-tag>{{ (record.chain_type || '').toUpperCase() }}</a-tag>
          </template>
          <template v-else-if="column.key === 'address'">
            <a-typography-text :ellipsis="{ tooltip: true }">
              {{ record.address }}
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
import { depositAddressAPI } from '../api'

const addresses = ref([])
const loading = ref(false)
const filters = ref({
  chain_type: undefined
})
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const columns = [
  {
    title: '用户ID',
    dataIndex: 'user_id',
    key: 'user_id',
    width: 100
  },
  {
    title: '链类型',
    dataIndex: 'chain_type',
    key: 'chain_type',
    width: 100
  },
  {
    title: '充值地址',
    dataIndex: 'address',
    key: 'address'
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

const loadAddresses = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.value.chain_type !== undefined) params.chain_type = filters.value.chain_type
    
    const res = await depositAddressAPI.getDepositAddresses(params)
    if (res.code === 200) {
      addresses.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载地址列表失败')
    }
  } catch (error) {
    message.error('加载地址列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadAddresses()
}

onMounted(() => {
  loadAddresses()
})
</script>

<style scoped>
.deposit-addresses {
  padding: 0;
}
</style>
