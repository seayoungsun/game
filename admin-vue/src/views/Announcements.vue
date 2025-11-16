<template>
  <div class="announcements">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">公告管理</span>
      </template>
      <template #extra>
        <a-button type="primary" @click="showCreateModal">新增公告</a-button>
      </template>

      <a-table
        :columns="columns"
        :data-source="announcements"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">
              {{ getTypeName(record.type) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'priority'">
            <a-tag :color="getPriorityColor(record.priority)">
              {{ getPriorityName(record.priority) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'default'">
              {{ record.status === 1 ? '已发布' : '已下架' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'target_users'">
            {{ record.target_users === 'all' ? '全部用户' : '指定用户' }}
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">编辑</a-button>
              <a-button type="link" size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑公告对话框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      @ok="handleSubmit"
      @cancel="handleCancel"
      :confirm-loading="submitting"
      width="800px"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        layout="vertical"
      >
        <a-form-item label="公告标题" name="title">
          <a-input v-model:value="formData.title" placeholder="请输入公告标题" />
        </a-form-item>
        <a-form-item label="公告内容" name="content">
          <a-textarea
            v-model:value="formData.content"
            placeholder="请输入公告内容"
            :rows="6"
          />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="公告类型" name="type">
              <a-select v-model:value="formData.type" placeholder="请选择类型">
                <a-select-option value="info">信息</a-select-option>
                <a-select-option value="success">成功</a-select-option>
                <a-select-option value="warning">警告</a-select-option>
                <a-select-option value="error">错误</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="优先级" name="priority">
              <a-select v-model:value="formData.priority" placeholder="请选择优先级">
                <a-select-option :value="0">普通</a-select-option>
                <a-select-option :value="1">重要</a-select-option>
                <a-select-option :value="2">紧急</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="状态" name="status">
              <a-select v-model:value="formData.status" placeholder="请选择状态">
                <a-select-option :value="1">已发布</a-select-option>
                <a-select-option :value="2">已下架</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="目标用户" name="target_users">
              <a-select v-model:value="formData.target_users" placeholder="请选择目标用户">
                <a-select-option value="all">全部用户</a-select-option>
                <a-select-option value="custom">指定用户</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="指定用户ID" v-if="formData.target_users === 'custom'">
          <a-input
            v-model:value="formData.target_users_custom"
            placeholder="请输入用户ID，多个用逗号分隔，如：1,2,3"
          />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="开始时间">
              <a-date-picker
                v-model:value="formData.start_time"
                show-time
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
                placeholder="选择开始时间（可选）"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="结束时间">
              <a-date-picker
                v-model:value="formData.end_time"
                show-time
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
                placeholder="选择结束时间（可选）"
              />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { DatePicker } from 'ant-design-vue'
import { messageAPI } from '../api'

const announcements = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const modalVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const currentId = ref(null)

const formData = ref({
  title: '',
  content: '',
  type: 'info',
  priority: 0,
  status: 1,
  target_users: 'all',
  target_users_custom: '',
  start_time: null,
  end_time: null
})

const formRules = {
  title: [{ required: true, message: '请输入公告标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入公告内容', trigger: 'blur' }]
}

const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80
  },
  {
    title: '标题',
    dataIndex: 'title',
    key: 'title',
    width: 200
  },
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
    width: 100
  },
  {
    title: '优先级',
    dataIndex: 'priority',
    key: 'priority',
    width: 100
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100
  },
  {
    title: '目标用户',
    dataIndex: 'target_users',
    key: 'target_users',
    width: 120
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

const modalTitle = computed(() => isEdit.value ? '编辑公告' : '新增公告')

const getTypeName = (type) => {
  const typeMap = {
    'info': '信息',
    'success': '成功',
    'warning': '警告',
    'error': '错误'
  }
  return typeMap[type] || type
}

const getTypeColor = (type) => {
  const colorMap = {
    'info': 'blue',
    'success': 'green',
    'warning': 'orange',
    'error': 'red'
  }
  return colorMap[type] || 'default'
}

const getPriorityName = (priority) => {
  const priorityMap = {
    0: '普通',
    1: '重要',
    2: '紧急'
  }
  return priorityMap[priority] || '普通'
}

const getPriorityColor = (priority) => {
  const colorMap = {
    0: 'default',
    1: 'orange',
    2: 'red'
  }
  return colorMap[priority] || 'default'
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

const loadAnnouncements = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.value.status) {
      params.status = filters.value.status
    }
    if (filters.value.search) {
      params.search = filters.value.search
    }
    
    const res = await messageAPI.getAnnouncements(params)
    if (res.code === 200) {
      announcements.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载公告列表失败')
    }
  } catch (error) {
    message.error('加载公告列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const filters = ref({
  status: '',
  search: ''
})

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadAnnouncements()
}

const showCreateModal = () => {
  isEdit.value = false
  currentId.value = null
  formData.value = {
    title: '',
    content: '',
    type: 'info',
    priority: 0,
    status: 1,
    target_users: 'all',
    target_users_custom: '',
    start_time: null,
    end_time: null
  }
  modalVisible.value = true
}

const handleEdit = (record) => {
  isEdit.value = true
  currentId.value = record.id
  formData.value = {
    title: record.title,
    content: record.content,
    type: record.type,
    priority: record.priority,
    status: record.status,
    target_users: record.target_users === 'all' ? 'all' : 'custom',
    target_users_custom: record.target_users === 'all' ? '' : record.target_users,
    start_time: record.start_time ? new Date(record.start_time * 1000) : null,
    end_time: record.end_time ? new Date(record.end_time * 1000) : null
  }
  modalVisible.value = true
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除公告"${record.title}"吗？`,
    onOk: async () => {
      try {
        const res = await messageAPI.deleteAnnouncement(record.id)
        if (res.code === 200) {
          message.success('删除成功')
          loadAnnouncements()
        } else {
          message.error(res.message || '删除失败')
        }
      } catch (error) {
        message.error('删除失败：' + (error.response?.data?.message || error.message))
      }
    }
  })
}

const handleSubmit = async () => {
  try {
    await formRef.value.validate()
  } catch (error) {
    return
  }
  
  submitting.value = true
  try {
    const submitData = {
      title: formData.value.title,
      content: formData.value.content,
      type: formData.value.type,
      priority: formData.value.priority,
      status: formData.value.status,
      target_users: formData.value.target_users === 'custom' 
        ? formData.value.target_users_custom 
        : 'all',
      start_time: formData.value.start_time ? Math.floor(new Date(formData.value.start_time).getTime() / 1000) : null,
      end_time: formData.value.end_time ? Math.floor(new Date(formData.value.end_time).getTime() / 1000) : null
    }

    if (isEdit.value) {
      const res = await messageAPI.updateAnnouncement(currentId.value, submitData)
      if (res.code === 200) {
        message.success('更新成功')
        modalVisible.value = false
        loadAnnouncements()
      } else {
        message.error(res.message || '更新失败')
      }
    } else {
      const res = await messageAPI.createAnnouncement(submitData)
      if (res.code === 200) {
        message.success('创建成功')
        modalVisible.value = false
        loadAnnouncements()
      } else {
        message.error(res.message || '创建失败')
      }
    }
  } catch (error) {
    message.error((isEdit.value ? '更新' : '创建') + '失败：' + (error.response?.data?.message || error.message))
  } finally {
    submitting.value = false
  }
}

const handleCancel = () => {
  modalVisible.value = false
  formRef.value?.resetFields()
}

onMounted(() => {
  loadAnnouncements()
})
</script>

<style scoped>
.announcements {
  padding: 0;
}
</style>

