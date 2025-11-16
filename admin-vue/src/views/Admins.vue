<template>
  <div class="admins">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">管理员管理</span>
      </template>
      <template #extra>
        <a-space>
          <a-input-search
            v-model:value="searchKeyword"
            placeholder="搜索用户名或昵称"
            style="width: 300px"
            @search="handleSearch"
            allow-clear
          />
          <a-button type="primary" @click="showCreateModal">新增管理员</a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="admins"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'error'">
              {{ record.status === 1 ? '正常' : '禁用' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'roles'">
            <a-tag v-for="role in record.roles" :key="role.id" style="margin: 2px" color="blue">
              {{ role.role_name }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'last_login_at'">
            {{ record.last_login_at ? formatTime(record.last_login_at) : '-' }}
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

    <!-- 创建/编辑管理员对话框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      @ok="handleSubmit"
      @cancel="handleCancel"
      :confirm-loading="submitting"
      width="600px"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        layout="vertical"
      >
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="formData.username"
            placeholder="请输入用户名"
            :disabled="isEdit"
          />
        </a-form-item>
        <a-form-item label="昵称" name="nickname">
          <a-input v-model:value="formData.nickname" placeholder="请输入昵称" />
        </a-form-item>
        <a-form-item label="邮箱" name="email">
          <a-input v-model:value="formData.email" placeholder="请输入邮箱" />
        </a-form-item>
        <a-form-item :label="isEdit ? '新密码（留空不修改）' : '密码'" name="password">
          <a-input-password
            v-model:value="formData.password"
            :placeholder="isEdit ? '留空则不修改密码' : '请输入密码'"
          />
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-radio-group v-model:value="formData.status">
            <a-radio :value="1">正常</a-radio>
            <a-radio :value="2">禁用</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="角色分配" name="role_ids">
          <a-select
            v-model:value="formData.role_ids"
            mode="multiple"
            placeholder="请选择角色"
            style="width: 100%"
            :options="roleOptions"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { adminAPI, roleAPI } from '../api'

const admins = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const modalVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const currentAdminId = ref(null)
const roleOptions = ref([])

const formData = ref({
  username: '',
  nickname: '',
  email: '',
  password: '',
  status: 1,
  role_ids: []
})

const formRules = computed(() => ({
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: isEdit.value
    ? []
    : [{ required: true, message: '请输入密码', trigger: 'blur' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}))

const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80
  },
  {
    title: '用户名',
    dataIndex: 'username',
    key: 'username',
    width: 150
  },
  {
    title: '昵称',
    dataIndex: 'nickname',
    key: 'nickname',
    width: 150
  },
  {
    title: '邮箱',
    dataIndex: 'email',
    key: 'email',
    width: 200
  },
  {
    title: '角色',
    key: 'roles',
    width: 200
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100
  },
  {
    title: '最后登录时间',
    dataIndex: 'last_login_at',
    key: 'last_login_at',
    width: 180
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

const modalTitle = computed(() => isEdit.value ? '编辑管理员' : '新增管理员')

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

const loadAdmins = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (searchKeyword.value) {
      params.search = searchKeyword.value
    }
    
    const res = await adminAPI.getAdmins(params)
    if (res.code === 200) {
      admins.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载管理员列表失败')
    }
  } catch (error) {
    message.error('加载管理员列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const loadRoles = async () => {
  try {
    const res = await roleAPI.getRoles({ page: 1, page_size: 1000 })
    if (res.code === 200) {
      roleOptions.value = (res.data.list || []).map(role => ({
        label: role.role_name,
        value: role.id
      }))
    }
  } catch (error) {
    console.error('加载角色列表失败:', error)
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadAdmins()
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadAdmins()
}

const showCreateModal = () => {
  isEdit.value = false
  currentAdminId.value = null
  formData.value = {
    username: '',
    nickname: '',
    email: '',
    password: '',
    status: 1,
    role_ids: []
  }
  modalVisible.value = true
}

const handleEdit = async (record) => {
  isEdit.value = true
  currentAdminId.value = record.id
  try {
    const res = await adminAPI.getAdmin(record.id)
    if (res.code === 200) {
      const admin = res.data
      formData.value = {
        username: admin.username,
        nickname: admin.nickname || '',
        email: admin.email || '',
        password: '',
        status: admin.status,
        role_ids: admin.roles?.map(r => r.id) || []
      }
      modalVisible.value = true
    } else {
      message.error(res.message || '获取管理员详情失败')
    }
  } catch (error) {
    message.error('获取管理员详情失败：' + (error.response?.data?.message || error.message))
  }
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除管理员"${record.username}"吗？`,
    onOk: async () => {
      try {
        const res = await adminAPI.deleteAdmin(record.id)
        if (res.code === 200) {
          message.success('删除成功')
          loadAdmins()
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
      nickname: formData.value.nickname,
      email: formData.value.email,
      status: formData.value.status,
      role_ids: formData.value.role_ids
    }
    
    // 只在创建时或编辑时提供了密码才添加密码字段
    if (!isEdit.value || formData.value.password) {
      submitData.password = formData.value.password
    }
    
    if (isEdit.value) {
      const res = await adminAPI.updateAdmin(currentAdminId.value, submitData)
      if (res.code === 200) {
        message.success('更新成功')
        modalVisible.value = false
        loadAdmins()
      } else {
        message.error(res.message || '更新失败')
      }
    } else {
      submitData.username = formData.value.username
      const res = await adminAPI.createAdmin(submitData)
      if (res.code === 200) {
        message.success('创建成功')
        modalVisible.value = false
        loadAdmins()
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
  loadAdmins()
  loadRoles()
})
</script>

<style scoped>
.admins {
  padding: 0;
}
</style>











