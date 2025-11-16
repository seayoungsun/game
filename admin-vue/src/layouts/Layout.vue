<template>
  <a-layout style="min-height: 100vh">
    <!-- 侧边栏 -->
    <a-layout-sider :width="250" theme="dark" v-model:collapsed="collapsed" :collapsible="true">
      <div class="logo">
        <h2 style="color: white; text-align: center; padding: 20px 0; margin: 0;">管理后台</h2>
      </div>
      <a-menu
        v-model:selectedKeys="selectedKeys"
        theme="dark"
        mode="inline"
        @select="handleMenuSelect"
      >
        <a-menu-item key="/dashboard">
          <template #icon>
            <DashboardOutlined />
          </template>
          <span>仪表盘</span>
        </a-menu-item>
        <a-menu-item key="/users">
          <template #icon>
            <UserOutlined />
          </template>
          <span>用户管理</span>
        </a-menu-item>
        <a-menu-item key="/recharge-orders">
          <template #icon>
            <WalletOutlined />
          </template>
          <span>充值订单</span>
        </a-menu-item>
        <a-menu-item key="/withdraw-orders">
          <template #icon>
            <CreditCardOutlined />
          </template>
          <span>提现订单</span>
        </a-menu-item>
        <a-menu-item key="/deposit-addresses">
          <template #icon>
            <KeyOutlined />
          </template>
          <span>充值地址</span>
        </a-menu-item>
        <a-menu-item key="/collection">
          <template #icon>
            <SyncOutlined />
          </template>
          <span>USDT归集</span>
        </a-menu-item>
        <a-sub-menu key="/system">
          <template #icon>
            <SettingOutlined />
          </template>
          <template #title>系统管理</template>
          <a-menu-item key="/roles">
            <template #icon>
              <UserSwitchOutlined />
            </template>
            <span>角色管理</span>
          </a-menu-item>
          <a-menu-item key="/admins">
            <template #icon>
              <TeamOutlined />
            </template>
            <span>管理员管理</span>
          </a-menu-item>
          <a-menu-item key="/operation-logs">
            <template #icon>
              <FileTextOutlined />
            </template>
            <span>操作日志</span>
          </a-menu-item>
          <a-menu-item key="/system-config">
            <template #icon>
              <ToolOutlined />
            </template>
            <span>系统设置</span>
          </a-menu-item>
        </a-sub-menu>
        <a-menu-item key="/announcements">
          <template #icon>
            <BellOutlined />
          </template>
          <span>公告管理</span>
        </a-menu-item>
        <a-menu-item key="/user-messages">
          <template #icon>
            <MessageOutlined />
          </template>
          <span>消息管理</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <!-- 主内容区 -->
    <a-layout>
      <a-layout-header style="background: #fff; padding: 0 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.1)">
        <div style="display: flex; justify-content: space-between; align-items: center; height: 100%">
          <div>
            <h2 style="margin: 0; font-size: 18px; font-weight: 600">{{ pageTitle }}</h2>
          </div>
          <div style="display: flex; align-items: center; gap: 16px">
            <span>{{ adminInfo }}</span>
            <a-button type="primary" danger @click="handleLogout">退出登录</a-button>
          </div>
        </div>
      </a-layout-header>

      <a-layout-content style="margin: 24px; padding: 24px; background: #fff; min-height: calc(100vh - 112px)">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined,
  UserOutlined,
  WalletOutlined,
  CreditCardOutlined,
  KeyOutlined,
  SyncOutlined,
  SettingOutlined,
  UserSwitchOutlined,
  TeamOutlined,
  FileTextOutlined,
  ToolOutlined,
  BellOutlined,
  MessageOutlined
} from '@ant-design/icons-vue'
import { authAPI, setToken } from '../api'

const router = useRouter()
const route = useRoute()

const collapsed = ref(false)
const selectedKeys = ref([route.path])
const adminInfo = ref('管理员')

const pageTitle = computed(() => {
  return route.meta.title || '管理后台'
})

// 监听路由变化
watch(() => route.path, (newPath) => {
  // 如果是系统管理子菜单，设置父菜单为选中状态
  if (newPath.startsWith('/roles') || newPath.startsWith('/admins')) {
    selectedKeys.value = [newPath]
  } else {
    selectedKeys.value = [newPath]
  }
})

// 检查登录状态
const checkAuth = async () => {
  const token = localStorage.getItem('token')
  if (token) {
    setToken(token)
    try {
      const res = await authAPI.getProfile()
      if (res.code === 200) {
        adminInfo.value = res.data.nickname || res.data.username
      } else {
        localStorage.removeItem('token')
        router.push('/login')
      }
    } catch (error) {
      localStorage.removeItem('token')
      router.push('/login')
    }
  } else {
    router.push('/login')
  }
}

// 菜单选择处理
const handleMenuSelect = ({ key }) => {
  router.push(key)
}

// 退出登录
const handleLogout = () => {
  localStorage.removeItem('token')
  setToken(null)
  router.push('/login')
  message.success('已退出登录')
}

onMounted(() => {
  checkAuth()
})
</script>

<style scoped>
.logo {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}
</style>

