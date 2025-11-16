<template>
  <div class="user-messages">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">消息管理</span>
      </template>
      <template #extra>
        <a-button type="primary" @click="showSendModal">发送消息</a-button>
      </template>

      <a-table
        :columns="columns"
        :data-source="messages"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'user_id'">
            <a-button type="link" @click="viewUser(record.user_id)">
              {{ record.user_id }}
            </a-button>
          </template>
          <template v-else-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">
              {{ getTypeName(record.type) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'is_read'">
            <a-tag :color="record.is_read ? 'success' : 'warning'">
              {{ record.is_read ? '已读' : '未读' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 发送消息对话框 -->
    <a-modal
      v-model:open="sendModalVisible"
      title="发送消息"
      @ok="handleSend"
      @cancel="handleCancelSend"
      :confirm-loading="sending"
      width="600px"
    >
      <a-form
        ref="sendFormRef"
        :model="sendFormData"
        :rules="sendFormRules"
        layout="vertical"
      >
        <a-form-item label="接收用户" name="user_ids">
          <a-select
            v-model:value="sendFormData.user_ids"
            mode="multiple"
            placeholder="请选择接收用户（可多选）"
            :options="userOptions"
            :filter-option="filterOption"
            show-search
            style="width: 100%"
          />
          <div style="margin-top: 8px; color: #999;">
            也可以手动输入用户ID，多个用逗号分隔
          </div>
        </a-form-item>
        <a-form-item label="消息类型" name="type">
          <a-select v-model:value="sendFormData.type" placeholder="请选择消息类型">
            <a-select-option value="info">信息</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="warning">警告</a-select-option>
            <a-select-option value="error">错误</a-select-option>
            <a-select-option value="system">系统</a-select-option>
            <a-select-option value="order">订单</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="消息标题" name="title">
          <a-input v-model:value="sendFormData.title" placeholder="请输入消息标题" />
        </a-form-item>
        <a-form-item label="消息内容" name="content">
          <a-textarea
            v-model:value="sendFormData.content"
            placeholder="请输入消息内容"
            :rows="4"
          />
        </a-form-item>
        <a-form-item label="关联订单号（可选）" name="related_id">
          <a-input v-model:value="sendFormData.related_id" placeholder="请输入订单号" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { messageAPI, userAPI } from '../api'

const messages = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const sendModalVisible = ref(false)
const sending = ref(false)
const sendFormRef = ref(null)
const userOptions = ref([])

const sendFormData = ref({
  user_ids: [],
  type: 'info',
  title: '',
  content: '',
  related_id: ''
})

const sendFormRules = {
  user_ids: [{ required: true, message: '请选择接收用户', trigger: 'change' }],
  title: [{ required: true, message: '请输入消息标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入消息内容', trigger: 'blur' }]
}

const filters = ref({
  user_id: '',
  type: '',
  is_read: ''
})

const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80
  },
  {
    title: '用户ID',
    dataIndex: 'user_id',
    key: 'user_id',
    width: 100
  },
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
    width: 100
  },
  {
    title: '标题',
    dataIndex: 'title',
    key: 'title',
    width: 200
  },
  {
    title: '内容',
    dataIndex: 'content',
    key: 'content',
    ellipsis: true
  },
  {
    title: '是否已读',
    dataIndex: 'is_read',
    key: 'is_read',
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
    width: 100,
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

const getTypeName = (type) => {
  const typeMap = {
    'info': '信息',
    'success': '成功',
    'warning': '警告',
    'error': '错误',
    'system': '系统',
    'order': '订单'
  }
  return typeMap[type] || type
}

const getTypeColor = (type) => {
  const colorMap = {
    'info': 'blue',
    'success': 'green',
    'warning': 'orange',
    'error': 'red',
    'system': 'purple',
    'order': 'cyan'
  }
  return colorMap[type] || 'default'
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

const loadMessages = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.value.user_id) {
      params.user_id = filters.value.user_id
    }
    if (filters.value.type) {
      params.type = filters.value.type
    }
    if (filters.value.is_read) {
      params.is_read = filters.value.is_read
    }
    
    const res = await messageAPI.getUserMessages(params)
    if (res.code === 200) {
      messages.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载消息列表失败')
    }
  } catch (error) {
    message.error('加载消息列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const loadUsers = async () => {
  try {
    const res = await userAPI.getUsers({ page: 1, page_size: 1000 })
    if (res.code === 200) {
      userOptions.value = (res.data.list || []).map(user => {
        const displayName = user.nickname || user.phone || `用户${user.id}`
        return {
          label: `${displayName} (ID: ${user.id})`,
          value: user.id
        }
      })
    }
  } catch (error) {
    console.error('加载用户列表失败:', error)
  }
}

const filterOption = (input, option) => {
  return option.label.toLowerCase().includes(input.toLowerCase())
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadMessages()
}

const showSendModal = () => {
  sendFormData.value = {
    user_ids: [],
    type: 'info',
    title: '',
    content: '',
    related_id: ''
  }
  sendModalVisible.value = true
}

const handleSend = async () => {
  try {
    await sendFormRef.value.validate()
  } catch (error) {
    return
  }
  
  sending.value = true
  try {
    const res = await messageAPI.sendUserMessage({
      user_ids: sendFormData.value.user_ids,
      type: sendFormData.value.type,
      title: sendFormData.value.title,
      content: sendFormData.value.content,
      related_id: sendFormData.value.related_id || undefined
    })
    
    if (res.code === 200) {
      message.success(`成功发送 ${res.data.count || 0} 条消息`)
      sendModalVisible.value = false
      loadMessages()
    } else {
      message.error(res.message || '发送失败')
    }
  } catch (error) {
    message.error('发送失败：' + (error.response?.data?.message || error.message))
  } finally {
    sending.value = false
  }
}

const handleCancelSend = () => {
  sendModalVisible.value = false
  sendFormRef.value?.resetFields()
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除这条消息吗？`,
    onOk: async () => {
      try {
        const res = await messageAPI.deleteUserMessage(record.id)
        if (res.code === 200) {
          message.success('删除成功')
          loadMessages()
        } else {
          message.error(res.message || '删除失败')
        }
      } catch (error) {
        message.error('删除失败：' + (error.response?.data?.message || error.message))
      }
    }
  })
}

const viewUser = (userId) => {
  // 可以跳转到用户详情页面
  window.open(`/users/${userId}`, '_blank')
}

onMounted(() => {
  loadMessages()
  loadUsers()
})
</script>

<style scoped>
.user-messages {
  padding: 0;
}
</style>

