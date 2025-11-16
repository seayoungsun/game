import { createRouter, createWebHistory } from 'vue-router'
import Layout from '../layouts/Layout.vue'
import Login from '../views/Login.vue'
import Dashboard from '../views/Dashboard.vue'
import Users from '../views/Users.vue'
import RechargeOrders from '../views/RechargeOrders.vue'
import WithdrawOrders from '../views/WithdrawOrders.vue'
import DepositAddresses from '../views/DepositAddresses.vue'
import Collection from '../views/Collection.vue'
import Roles from '../views/Roles.vue'
import Admins from '../views/Admins.vue'
import OperationLogs from '../views/OperationLogs.vue'
import SystemConfig from '../views/SystemConfig.vue'
import Announcements from '../views/Announcements.vue'
import UserMessages from '../views/UserMessages.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: Dashboard,
        meta: { title: '仪表盘', requiresAuth: true }
      },
      {
        path: 'users',
        name: 'Users',
        component: Users,
        meta: { title: '用户管理', requiresAuth: true }
      },
      {
        path: 'recharge-orders',
        name: 'RechargeOrders',
        component: RechargeOrders,
        meta: { title: '充值订单', requiresAuth: true }
      },
      {
        path: 'withdraw-orders',
        name: 'WithdrawOrders',
        component: WithdrawOrders,
        meta: { title: '提现订单', requiresAuth: true }
      },
      {
        path: 'deposit-addresses',
        name: 'DepositAddresses',
        component: DepositAddresses,
        meta: { title: '充值地址', requiresAuth: true }
      },
      {
        path: 'collection',
        name: 'Collection',
        component: Collection,
        meta: { title: 'USDT归集', requiresAuth: true }
      },
      {
        path: 'roles',
        name: 'Roles',
        component: Roles,
        meta: { title: '角色管理', requiresAuth: true }
      },
      {
        path: 'admins',
        name: 'Admins',
        component: Admins,
        meta: { title: '管理员管理', requiresAuth: true }
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: OperationLogs,
        meta: { title: '操作日志', requiresAuth: true }
      },
      {
        path: 'system-config',
        name: 'SystemConfig',
        component: SystemConfig,
        meta: { title: '系统设置', requiresAuth: true }
      },
      {
        path: 'announcements',
        name: 'Announcements',
        component: Announcements,
        meta: { title: '公告管理', requiresAuth: true }
      },
      {
        path: 'user-messages',
        name: 'UserMessages',
        component: UserMessages,
        meta: { title: '消息管理', requiresAuth: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
