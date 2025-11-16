<template>
  <div class="users">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">用户列表</span>
      </template>
      <template #extra>
        <a-input-search
          v-model:value="searchKeyword"
          placeholder="搜索手机号或昵称"
          style="width: 300px"
          @search="handleSearch"
          allow-clear
        />
      </template>

      <a-table
        :columns="columns"
        :data-source="users"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.uid"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'balance'">
            {{ parseFloat(record.balance || 0).toFixed(2) }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'error'">
              {{ record.status === 1 ? '正常' : '封禁' }}
            </a-tag>
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
import { userAPI } from '../api'

const users = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const columns = [
  {
    title: 'ID',
    dataIndex: 'uid',
    key: 'uid',
    width: 100
  },
  {
    title: '手机号',
    dataIndex: 'phone',
    key: 'phone',
    width: 150
  },
  {
    title: '昵称',
    dataIndex: 'nickname',
    key: 'nickname',
    width: 150
  },
  {
    title: '余额',
    dataIndex: 'balance',
    key: 'balance',
    width: 120
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100
  },
  {
    title: '注册时间',
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

const loadUsers = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (searchKeyword.value) {
      params.search = searchKeyword.value
    }
    
    const res = await userAPI.getUsers(params)
    if (res.code === 200) {
      users.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载用户列表失败')
    }
  } catch (error) {
    message.error('加载用户列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadUsers()
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadUsers()
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.users {
  padding: 0;
}
</style>
