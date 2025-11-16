<template>
  <div class="collection">
    <a-row :gutter="[20, 20]">
      <a-col :xs="24" :sm="24" :md="12">
        <a-card title="单用户归集" :bordered="false">
          <a-form
            :model="singleForm"
            :rules="singleRules"
            ref="singleFormRef"
            layout="vertical"
          >
            <a-form-item label="用户ID" name="user_id">
              <a-input-number
                v-model:value="singleForm.user_id"
                :min="1"
                style="width: 100%"
                placeholder="请输入用户ID"
              />
            </a-form-item>
            <a-form-item label="链类型" name="chain_type">
              <a-select
                v-model:value="singleForm.chain_type"
                placeholder="请选择链类型"
                style="width: 100%"
              >
                <a-select-option value="trc20">TRC20</a-select-option>
                <a-select-option value="erc20">ERC20</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item>
              <a-button
                type="primary"
                @click="handleSingleCollect"
                :loading="singleLoading"
                block
              >
                执行归集
              </a-button>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <a-col :xs="24" :sm="24" :md="12">
        <a-card title="批量归集" :bordered="false">
          <a-form
            :model="batchForm"
            :rules="batchRules"
            ref="batchFormRef"
            layout="vertical"
          >
            <a-form-item label="链类型" name="chain_type">
              <a-select
                v-model:value="batchForm.chain_type"
                placeholder="请选择链类型"
                style="width: 100%"
              >
                <a-select-option value="trc20">TRC20</a-select-option>
                <a-select-option value="erc20">ERC20</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="归集数量" name="limit">
              <a-input-number
                v-model:value="batchForm.limit"
                :min="1"
                :max="100"
                style="width: 100%"
                placeholder="请输入归集数量"
              />
            </a-form-item>
            <a-form-item>
              <a-button
                type="primary"
                @click="handleBatchCollect"
                :loading="batchLoading"
                block
              >
                批量归集
              </a-button>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>
    </a-row>

    <a-card
      v-if="result"
      :title="result.title"
      :bordered="false"
      style="margin-top: 20px"
    >
      <a-alert
        :message="result.title"
        :description="result.description"
        :type="result.type"
        show-icon
      >
        <template v-if="result.tx_hash" #action>
          <a-typography-text copyable>{{ result.tx_hash }}</a-typography-text>
        </template>
      </a-alert>
    </a-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { collectionAPI } from '../api'

const singleForm = ref({
  user_id: null,
  chain_type: 'trc20'
})
const singleRules = {
  user_id: [{ required: true, message: '请输入用户ID', trigger: 'blur' }],
  chain_type: [{ required: true, message: '请选择链类型', trigger: 'change' }]
}
const singleFormRef = ref(null)
const singleLoading = ref(false)

const batchForm = ref({
  chain_type: 'trc20',
  limit: 10
})
const batchRules = {
  chain_type: [{ required: true, message: '请选择链类型', trigger: 'change' }],
  limit: [{ required: true, message: '请输入归集数量', trigger: 'blur' }]
}
const batchFormRef = ref(null)
const batchLoading = ref(false)

const result = ref(null)

const handleSingleCollect = async () => {
  if (!singleFormRef.value) return
  
  try {
    await singleFormRef.value.validate()
  } catch (error) {
    return
  }
  
  singleLoading.value = true
  result.value = null
  
  try {
    const res = await collectionAPI.collectUSDT(singleForm.value)
    if (res.code === 200) {
      result.value = {
        type: 'success',
        title: '归集成功',
        description: `用户 ${singleForm.value.user_id} 的 ${singleForm.value.chain_type.toUpperCase()} USDT 归集成功`,
        tx_hash: res.data?.tx_hash
      }
      message.success('归集成功')
    } else {
      result.value = {
        type: 'error',
        title: '归集失败',
        description: res.message || '归集失败'
      }
      message.error(res.message || '归集失败')
    }
  } catch (error) {
    result.value = {
      type: 'error',
      title: '归集失败',
      description: error.response?.data?.message || error.message || '归集失败'
    }
    message.error('归集失败：' + (error.response?.data?.message || error.message))
  } finally {
    singleLoading.value = false
  }
}

const handleBatchCollect = async () => {
  if (!batchFormRef.value) return
  
  try {
    await batchFormRef.value.validate()
  } catch (error) {
    return
  }
  
  batchLoading.value = true
  result.value = null
  
  try {
    const res = await collectionAPI.batchCollectUSDT(batchForm.value)
    if (res.code === 200) {
      result.value = {
        type: 'success',
        title: '批量归集成功',
        description: `已成功提交批量归集任务：${batchForm.value.chain_type.toUpperCase()}，数量：${batchForm.value.limit}`
      }
      message.success('批量归集成功')
    } else {
      result.value = {
        type: 'error',
        title: '批量归集失败',
        description: res.message || '批量归集失败'
      }
      message.error(res.message || '批量归集失败')
    }
  } catch (error) {
    result.value = {
      type: 'error',
      title: '批量归集失败',
      description: error.response?.data?.message || error.message || '批量归集失败'
    }
    message.error('批量归集失败：' + (error.response?.data?.message || error.message))
  } finally {
    batchLoading.value = false
  }
}
</script>

<style scoped>
.collection {
  padding: 0;
}
</style>
