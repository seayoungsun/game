<template>
  <div class="login-container">
    <a-card class="login-card" :bordered="false">
      <template #title>
        <div style="text-align: center; font-size: 24px; font-weight: 600">管理员登录</div>
      </template>
      <a-form
        :model="loginForm"
        :rules="loginRules"
        @finish="handleLogin"
        layout="vertical"
        size="large"
      >
        <a-form-item label="用户名" name="username">
          <a-input
            v-model:value="loginForm.username"
            placeholder="请输入用户名"
            @pressEnter="handleLogin"
          >
            <template #prefix>
              <UserOutlined />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="密码" name="password">
          <a-input-password
            v-model:value="loginForm.password"
            placeholder="请输入密码"
            @pressEnter="handleLogin"
          >
            <template #prefix>
              <LockOutlined />
            </template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button
            type="primary"
            html-type="submit"
            :loading="loading"
            block
            size="large"
          >
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined } from '@ant-design/icons-vue'
import { authAPI, setToken } from '../api'

const router = useRouter()

const loading = ref(false)
const loginForm = ref({
  username: '',
  password: ''
})

const loginRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const handleLogin = async () => {
  loading.value = true
  try {
    const res = await authAPI.login(loginForm.value.username, loginForm.value.password)
    if (res.code === 200) {
      localStorage.setItem('token', res.data.token)
      setToken(res.data.token)
      message.success('登录成功')
      router.push('/dashboard')
    } else {
      message.error(res.message || '登录失败')
    }
  } catch (error) {
    message.error('登录失败：' + (error.response?.data?.message || error.message))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 420px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

:deep(.ant-card-body) {
  padding: 32px;
}

:deep(.ant-form-item-label > label) {
  font-weight: 500;
}
</style>











