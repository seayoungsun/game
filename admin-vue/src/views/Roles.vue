<template>
  <div class="roles">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">角色管理</span>
      </template>
      <template #extra>
        <a-space>
          <a-input-search
            v-model:value="searchKeyword"
            placeholder="搜索角色名称或代码"
            style="width: 300px"
            @search="handleSearch"
            allow-clear
          />
          <a-button type="primary" @click="showCreateModal">新增角色</a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data-source="roles"
        :loading="loading"
        :pagination="paginationConfig"
        :row-key="(record) => record.id"
        @change="handleTableChange"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'error'">
              {{ record.status === 1 ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'permissions'">
            <a-tag v-for="perm in record.permissions" :key="perm.id" style="margin: 2px">
              {{ perm.permission_name }}
            </a-tag>
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

    <!-- 创建/编辑角色对话框 -->
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
        <a-form-item label="角色名称" name="role_name">
          <a-input v-model:value="formData.role_name" placeholder="请输入角色名称" />
        </a-form-item>
        <a-form-item label="角色代码" name="role_code">
          <a-input
            v-model:value="formData.role_code"
            placeholder="请输入角色代码（如：admin）"
            :disabled="isEdit"
          />
        </a-form-item>
        <a-form-item label="角色描述" name="description">
          <a-textarea
            v-model:value="formData.description"
            placeholder="请输入角色描述"
            :rows="3"
          />
        </a-form-item>
        <a-form-item label="权限配置" name="permission_codes">
          <a-tree-select
            v-model:value="formData.permission_codes"
            :tree-data="permissionTree"
            tree-checkable
            :show-checked-strategy="TreeSelect.SHOW_PARENT"
            placeholder="请选择权限"
            style="width: 100%"
            :max-tag-count="10"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal, TreeSelect } from 'ant-design-vue'
import { roleAPI, permissionAPI } from '../api'

const roles = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const modalVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const currentRoleId = ref(null)

const formData = ref({
  role_name: '',
  role_code: '',
  description: '',
  permission_codes: []
})

const formRules = {
  role_name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  role_code: [{ required: true, message: '请输入角色代码', trigger: 'blur' }]
}

const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80
  },
  {
    title: '角色名称',
    dataIndex: 'role_name',
    key: 'role_name',
    width: 150
  },
  {
    title: '角色代码',
    dataIndex: 'role_code',
    key: 'role_code',
    width: 150
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
    width: 200
  },
  {
    title: '权限',
    key: 'permissions',
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

const permissionTree = ref([])
const modalTitle = computed(() => isEdit.value ? '编辑角色' : '新增角色')

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

const loadRoles = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (searchKeyword.value) {
      params.search = searchKeyword.value
    }
    
    const res = await roleAPI.getRoles(params)
    if (res.code === 200) {
      roles.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      message.error(res.message || '加载角色列表失败')
    }
  } catch (error) {
    message.error('加载角色列表失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}

const loadPermissions = async () => {
  try {
    const res = await permissionAPI.getAllPermissions()
    if (res.code === 200) {
      // 将权限列表转换为树形结构（扁平结构）
      const permissions = res.data || []
      const tree = permissions.map(perm => ({
        title: `${perm.permission_name} (${perm.permission_code})`,
        value: perm.permission_code,
        key: perm.permission_code
      }))
      permissionTree.value = tree
    }
  } catch (error) {
    console.error('加载权限列表失败:', error)
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadRoles()
}

const handleTableChange = (pagination) => {
  currentPage.value = pagination.current
  pageSize.value = pagination.pageSize
  loadRoles()
}

const showCreateModal = () => {
  isEdit.value = false
  currentRoleId.value = null
  formData.value = {
    role_name: '',
    role_code: '',
    description: '',
    permission_codes: []
  }
  modalVisible.value = true
}

const handleEdit = async (record) => {
  isEdit.value = true
  currentRoleId.value = record.id
  try {
    const res = await roleAPI.getRole(record.id)
    if (res.code === 200) {
      const role = res.data
      formData.value = {
        role_name: role.role_name,
        role_code: role.role_code,
        description: role.description || '',
        permission_codes: role.permissions?.map(p => p.permission_code) || []
      }
      modalVisible.value = true
    } else {
      message.error(res.message || '获取角色详情失败')
    }
  } catch (error) {
    message.error('获取角色详情失败：' + (error.response?.data?.message || error.message))
  }
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除角色"${record.role_name}"吗？`,
    onOk: async () => {
      try {
        const res = await roleAPI.deleteRole(record.id)
        if (res.code === 200) {
          message.success('删除成功')
          loadRoles()
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
    if (isEdit.value) {
      const res = await roleAPI.updateRole(currentRoleId.value, formData.value)
      if (res.code === 200) {
        message.success('更新成功')
        modalVisible.value = false
        loadRoles()
      } else {
        message.error(res.message || '更新失败')
      }
    } else {
      const res = await roleAPI.createRole(formData.value)
      if (res.code === 200) {
        message.success('创建成功')
        modalVisible.value = false
        loadRoles()
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
  loadRoles()
  loadPermissions()
})
</script>

<style scoped>
.roles {
  padding: 0;
}
</style>

