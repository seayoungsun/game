import axios from 'axios'
import { message } from 'ant-design-vue'

// 开发环境使用相对路径（通过Vite代理），生产环境使用环境变量
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || (import.meta.env.DEV ? '/api' : 'http://localhost:8082/api')

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response) {
      if (error.response.status === 401) {
        localStorage.removeItem('token')
        window.location.href = '/login'
      } else if (error.response.status === 403) {
        message.error('权限不足')
      } else {
        message.error(error.response.data?.message || '请求失败')
      }
    } else {
      message.error('网络错误')
    }
    return Promise.reject(error)
  }
)

// 设置Token
export const setToken = (token) => {
  if (token) {
    localStorage.setItem('token', token)
    api.defaults.headers.Authorization = `Bearer ${token}`
  } else {
    localStorage.removeItem('token')
    delete api.defaults.headers.Authorization
  }
}

// 管理员认证API
export const authAPI = {
  // 管理员登录
  login: (username, password) => {
    return api.post('/v1/auth/login', { username, password })
  },
  // 获取管理员信息
  getProfile: () => {
    return api.get('/v1/admin/profile')
  },
  // 获取权限列表
  getPermissions: () => {
    return api.get('/v1/admin/permissions')
  }
}

// 用户相关API
export const userAPI = {
  // 获取用户列表
  getUsers: (params) => {
    return api.get('/v1/admin/users', { params })
  },
  // 获取用户详情
  getUser: (id) => {
    return api.get(`/v1/admin/users/${id}`)
  },
  // 更新用户
  updateUser: (id, data) => {
    return api.put(`/v1/admin/users/${id}`, data)
  },
  // 获取仪表盘统计
  getDashboardStats: () => {
    return api.get('/v1/admin/dashboard/stats')
  },
  // 获取仪表盘趋势数据
  getDashboardTrends: () => {
    return api.get('/v1/admin/dashboard/trends')
  }
}

// 充值订单API
export const rechargeAPI = {
  // 获取充值订单列表
  getRechargeOrders: (params) => {
    return api.get('/v1/admin/recharge-orders', { params })
  },
  // 获取充值订单详情
  getRechargeOrder: (orderId) => {
    return api.get(`/v1/admin/recharge-orders/${orderId}`)
  }
}

// 提现订单API
export const withdrawAPI = {
  // 获取提现订单列表
  getWithdrawOrders: (params) => {
    return api.get('/v1/admin/withdraw-orders', { params })
  },
  // 获取提现订单详情
  getWithdrawOrder: (orderId) => {
    return api.get(`/v1/admin/withdraw-orders/${orderId}`)
  },
  // 审核提现订单
  auditWithdrawOrder: (orderId, data) => {
    return api.post(`/v1/admin/withdraw-orders/${orderId}/audit`, data)
  }
}

// 充值地址API
export const depositAddressAPI = {
  // 获取充值地址列表
  getDepositAddresses: (params) => {
    return api.get('/v1/admin/deposit-addresses', { params })
  }
}

// USDT归集API
export const collectionAPI = {
  // 单用户归集
  collectUSDT: (data) => {
    return api.post('/v1/admin/payments/collect', data)
  },
  // 批量归集
  batchCollectUSDT: (data) => {
    return api.post('/v1/admin/payments/batch-collect', data)
  }
}

// 角色管理API
export const roleAPI = {
  // 获取角色列表
  getRoles: (params) => {
    return api.get('/v1/admin/roles', { params })
  },
  // 获取角色详情
  getRole: (id) => {
    return api.get(`/v1/admin/roles/${id}`)
  },
  // 创建角色
  createRole: (data) => {
    return api.post('/v1/admin/roles', data)
  },
  // 更新角色
  updateRole: (id, data) => {
    return api.put(`/v1/admin/roles/${id}`, data)
  },
  // 删除角色
  deleteRole: (id) => {
    return api.delete(`/v1/admin/roles/${id}`)
  }
}

// 权限管理API
export const permissionAPI = {
  // 获取所有权限列表
  getAllPermissions: () => {
    return api.get('/v1/admin/permissions/all')
  }
}

// 管理员管理API
export const adminAPI = {
  // 获取管理员列表
  getAdmins: (params) => {
    return api.get('/v1/admin/admins', { params })
  },
  // 获取管理员详情
  getAdmin: (id) => {
    return api.get(`/v1/admin/admins/${id}`)
  },
  // 创建管理员
  createAdmin: (data) => {
    return api.post('/v1/admin/admins', data)
  },
  // 更新管理员
  updateAdmin: (id, data) => {
    return api.put(`/v1/admin/admins/${id}`, data)
  },
  // 删除管理员
  deleteAdmin: (id) => {
    return api.delete(`/v1/admin/admins/${id}`)
  }
}

// 操作日志API
export const operationLogAPI = {
  // 获取操作日志列表
  getOperationLogs: (params) => {
    return api.get('/v1/admin/operation-logs', { params })
  },
  // 获取操作日志详情
  getOperationLog: (id) => {
    return api.get(`/v1/admin/operation-logs/${id}`)
  },
  // 删除操作日志
  deleteOperationLog: (id) => {
    return api.delete(`/v1/admin/operation-logs/${id}`)
  },
  // 批量删除操作日志
  batchDeleteOperationLogs: (ids) => {
    return api.post('/v1/admin/operation-logs/batch-delete', { ids })
  },
  // 清理旧日志
  cleanOldLogs: (days) => {
    return api.post('/v1/admin/operation-logs/clean', { days })
  }
}

// 系统设置API
export const systemConfigAPI = {
  // 获取系统配置列表
  getSystemConfigs: (params) => {
    return api.get('/v1/admin/system-configs', { params })
  },
  // 获取配置分组列表
  getSystemConfigGroups: () => {
    return api.get('/v1/admin/system-configs/groups')
  },
  // 获取单个配置
  getSystemConfig: (key) => {
    return api.get(`/v1/admin/system-configs/${key}`)
  },
  // 创建配置
  createSystemConfig: (data) => {
    return api.post('/v1/admin/system-configs', data)
  },
  // 更新配置
  updateSystemConfig: (key, data) => {
    return api.put(`/v1/admin/system-configs/${key}`, data)
  },
  // 删除配置
  deleteSystemConfig: (key) => {
    return api.delete(`/v1/admin/system-configs/${key}`)
  }
}

// 消息管理API
export const messageAPI = {
  // 公告管理
  // 获取公告列表
  getAnnouncements: (params) => {
    return api.get('/v1/admin/messages/announcements', { params })
  },
  // 获取公告详情
  getAnnouncement: (id) => {
    return api.get(`/v1/admin/messages/announcements/${id}`)
  },
  // 创建公告
  createAnnouncement: (data) => {
    return api.post('/v1/admin/messages/announcements', data)
  },
  // 更新公告
  updateAnnouncement: (id, data) => {
    return api.put(`/v1/admin/messages/announcements/${id}`, data)
  },
  // 删除公告
  deleteAnnouncement: (id) => {
    return api.delete(`/v1/admin/messages/announcements/${id}`)
  },
  
  // 用户消息管理
  // 获取用户消息列表
  getUserMessages: (params) => {
    return api.get('/v1/admin/messages/user-messages', { params })
  },
  // 发送用户消息
  sendUserMessage: (data) => {
    return api.post('/v1/admin/messages/user-messages/send', data)
  },
  // 删除用户消息
  deleteUserMessage: (id) => {
    return api.delete(`/v1/admin/messages/user-messages/${id}`)
  },
  // 批量删除用户消息
  batchDeleteUserMessages: (ids) => {
    return api.post('/v1/admin/messages/user-messages/batch-delete', { ids })
  }
}

export default api
