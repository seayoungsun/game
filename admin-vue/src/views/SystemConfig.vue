<template>
  <div class="system-config">
    <a-card :bordered="false">
      <template #title>
        <span style="font-size: 16px; font-weight: 600">系统设置</span>
      </template>
      <template #extra>
        <a-space>
          <a-select
            v-model:value="selectedGroup"
            placeholder="选择配置分组"
            style="width: 200px"
            @change="handleGroupChange"
          >
            <a-select-option value="">全部</a-select-option>
            <a-select-option v-for="group in groups" :key="group" :value="group">
              {{ getGroupName(group) }}
            </a-select-option>
          </a-select>
          <a-button type="primary" @click="showCreateModal">新增配置</a-button>
        </a-space>
      </template>

      <!-- 按分组展示配置 -->
      <div v-for="group in filteredGroups" :key="group" style="margin-bottom: 24px">
        <a-card :title="getGroupName(group)" :bordered="true" size="small">
          <a-table
            :columns="columns"
            :data-source="getConfigsByGroup(group)"
            :pagination="false"
            :row-key="(record) => record.id"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'config_key'">
                <strong>{{ record.config_key }}</strong>
                <div style="color: #999; font-size: 12px; margin-top: 4px;">
                  {{ record.description || '-' }}
                </div>
              </template>
              <template v-else-if="column.key === 'config_type'">
                <a-tag>{{ record.config_type }}</a-tag>
              </template>
              <template v-else-if="column.key === 'is_public'">
                <a-tag :color="record.is_public ? 'success' : 'default'">
                  {{ record.is_public ? '公开' : '私有' }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'config_value'">
                <span v-if="record.config_type === 'bool'">
                  <a-tag :color="record.config_value === 'true' ? 'success' : 'default'">
                    {{ record.config_value === 'true' ? '是' : '否' }}
                  </a-tag>
                </span>
                <span v-else-if="record.config_type === 'json'">
                  <pre style="margin: 0; max-width: 300px; overflow: auto; white-space: pre-wrap;">{{ formatJSON(record.config_value) }}</pre>
                </span>
                <span v-else style="max-width: 250px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ record.config_value || '-' }}</span>
              </template>
              <template v-else-if="column.key === 'updated_at'">
                {{ formatTime(record.updated_at) }}
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
      </div>
    </a-card>

    <!-- 创建/编辑配置对话框 -->
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
        <a-form-item label="配置键" name="config_key">
          <a-input
            v-model:value="formData.config_key"
            placeholder="请输入配置键（如：site_name）"
            :disabled="isEdit"
          />
        </a-form-item>
        <a-form-item label="配置值" name="config_value">
          <a-textarea
            v-if="formData.config_type === 'json'"
            v-model:value="formData.config_value"
            placeholder="请输入JSON格式的配置值"
            :rows="4"
          />
          <a-input-number
            v-else-if="formData.config_type === 'int' || formData.config_type === 'float'"
            v-model:value="formData.config_value"
            :style="{ width: '100%' }"
            :precision="formData.config_type === 'float' ? 2 : 0"
          />
          <a-switch
            v-else-if="formData.config_type === 'bool'"
            v-model:checked="formData.config_value"
            checked-children="是"
            un-checked-children="否"
          />
          <a-input
            v-else
            v-model:value="formData.config_value"
            placeholder="请输入配置值"
          />
        </a-form-item>
        <a-form-item label="配置类型" name="config_type">
          <a-select v-model:value="formData.config_type" placeholder="请选择配置类型">
            <a-select-option value="string">字符串</a-select-option>
            <a-select-option value="int">整数</a-select-option>
            <a-select-option value="float">浮点数</a-select-option>
            <a-select-option value="bool">布尔值</a-select-option>
            <a-select-option value="json">JSON</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="配置分组" name="group_name">
          <a-select v-model:value="formData.group_name" placeholder="请选择配置分组">
            <a-select-option value="site">站点设置</a-select-option>
            <a-select-option value="payment">支付设置</a-select-option>
            <a-select-option value="game">游戏设置</a-select-option>
            <a-select-option value="system">系统设置</a-select-option>
            <a-select-option value="default">默认</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="配置说明" name="description">
          <a-textarea
            v-model:value="formData.description"
            placeholder="请输入配置说明"
            :rows="2"
          />
        </a-form-item>
        <a-form-item label="是否公开" name="is_public">
          <a-switch v-model:checked="formData.is_public" />
          <span style="margin-left: 8px; color: #999;">
            公开的配置可以通过API获取，私有的配置仅管理员可见
          </span>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { systemConfigAPI } from '../api'

const configs = ref([])
const groups = ref([])
const selectedGroup = ref('')
const modalVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const formRef = ref(null)
const currentConfigKey = ref('')

const formData = ref({
  config_key: '',
  config_value: '',
  config_type: 'string',
  group_name: 'default',
  description: '',
  is_public: false
})

const formRules = {
  config_key: [{ required: true, message: '请输入配置键', trigger: 'blur' }],
  config_type: [{ required: true, message: '请选择配置类型', trigger: 'change' }],
  group_name: [{ required: true, message: '请选择配置分组', trigger: 'change' }]
}

const columns = [
  {
    title: '配置键',
    dataIndex: 'config_key',
    key: 'config_key',
    width: 200
  },
  {
    title: '配置值',
    dataIndex: 'config_value',
    key: 'config_value',
    width: 300
  },
  {
    title: '类型',
    dataIndex: 'config_type',
    key: 'config_type',
    width: 100
  },
  {
    title: '是否公开',
    dataIndex: 'is_public',
    key: 'is_public',
    width: 100
  },
  {
    title: '更新时间',
    dataIndex: 'updated_at',
    key: 'updated_at',
    width: 180
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right'
  }
]

const modalTitle = computed(() => isEdit.value ? '编辑配置' : '新增配置')

const filteredGroups = computed(() => {
  if (selectedGroup.value) {
    return [selectedGroup.value]
  }
  return groups.value
})

const getGroupName = (group) => {
  const groupMap = {
    'site': '站点设置',
    'payment': '支付设置',
    'game': '游戏设置',
    'system': '系统设置',
    'default': '默认配置'
  }
  return groupMap[group] || group
}

const getConfigsByGroup = (group) => {
  return configs.value.filter(c => c.group_name === group)
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

const formatJSON = (str) => {
  if (!str) return '-'
  try {
    const obj = JSON.parse(str)
    return JSON.stringify(obj, null, 2)
  } catch {
    return str
  }
}

const loadConfigs = async () => {
  try {
    const res = await systemConfigAPI.getSystemConfigs()
    if (res.code === 200) {
      configs.value = res.data || []
    } else {
      message.error(res.message || '加载配置失败')
    }
  } catch (error) {
    message.error('加载配置失败：' + (error.response?.data?.message || error.message))
  }
}

const loadGroups = async () => {
  try {
    const res = await systemConfigAPI.getSystemConfigGroups()
    if (res.code === 200) {
      groups.value = res.data || []
    }
  } catch (error) {
    console.error('加载配置分组失败:', error)
  }
}

const handleGroupChange = () => {
  // 分组切换时重新加载（如果需要）
}

const showCreateModal = () => {
  isEdit.value = false
  currentConfigKey.value = ''
  formData.value = {
    config_key: '',
    config_value: '',
    config_type: 'string',
    group_name: 'default',
    description: '',
    is_public: false
  }
  modalVisible.value = true
}

const handleEdit = (record) => {
  isEdit.value = true
  currentConfigKey.value = record.config_key
  formData.value = {
    config_key: record.config_key,
    config_value: record.config_type === 'bool' ? record.config_value === 'true' : record.config_value,
    config_type: record.config_type,
    group_name: record.group_name,
    description: record.description || '',
    is_public: record.is_public
  }
  modalVisible.value = true
}

const handleDelete = (record) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除配置"${record.config_key}"吗？`,
    onOk: async () => {
      try {
        const res = await systemConfigAPI.deleteSystemConfig(record.config_key)
        if (res.code === 200) {
          message.success('删除成功')
          loadConfigs()
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
      config_value: formData.value.config_type === 'bool' 
        ? (formData.value.config_value ? 'true' : 'false')
        : formData.value.config_value,
      config_type: formData.value.config_type,
      group_name: formData.value.group_name,
      description: formData.value.description,
      is_public: formData.value.is_public
    }

    if (isEdit.value) {
      const res = await systemConfigAPI.updateSystemConfig(currentConfigKey.value, submitData)
      if (res.code === 200) {
        message.success('更新成功')
        modalVisible.value = false
        loadConfigs()
      } else {
        message.error(res.message || '更新失败')
      }
    } else {
      submitData.config_key = formData.value.config_key
      const res = await systemConfigAPI.createSystemConfig(submitData)
      if (res.code === 200) {
        message.success('创建成功')
        modalVisible.value = false
        loadConfigs()
        loadGroups()
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
  loadConfigs()
  loadGroups()
})
</script>

<style scoped>
.system-config {
  padding: 0;
}
</style>

