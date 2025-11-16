# 管理后台综合文档（系统说明 + 部署指南 + 功能清单）

> 本文合并自：`admin_system.md`、`admin_setup_guide.md`、`admin_features.md`。用于统一查阅管理后台的架构说明、部署步骤、功能清单与权限说明。

---

## 一、系统说明（原 admin_system.md）

### 架构设计

管理后台采用**独立应用**架构，与前端用户API完全分离：

```
apps/
├── api/          # 前端用户API（端口8080）
├── game-server/  # 游戏服务器（端口8081）
└── admin/        # 管理后台API（端口8082）
```

### RBAC权限系统

#### 数据模型

1. admins - 管理员表（独立于users表）
2. admin_roles - 角色表
3. admin_permissions - 权限表
4. admin_role_relations - 管理员角色关联表
5. role_permission_relations - 角色权限关联表

#### 权限代码

采用 `资源:操作` 格式，例如：
- `admin:dashboard:view` - 查看仪表盘
- `admin:users:list` - 查看用户列表
- `admin:users:update` - 更新用户信息
- `admin:orders:audit` - 审核订单
- `admin:payments:collect` - USDT归集

#### 默认角色

- super_admin - 超级管理员（拥有所有权限）
- admin - 管理员（除系统管理外的所有权限）
- operator - 运营（用户管理和订单查看权限）
- auditor - 审核员（订单审核权限）

### API接口（概览）

#### 认证接口
- `POST /api/v1/auth/login` - 管理员登录
- `POST /api/v1/auth/logout` - 退出登录（需要Token）

#### 管理员信息
- `GET /api/v1/admin/profile` - 获取当前管理员信息
- `GET /api/v1/admin/permissions` - 获取权限列表

#### 功能接口（需要对应权限）
- `GET /api/v1/admin/dashboard/stats` - 仪表盘统计（需要 `admin:dashboard:view`）
- `GET /api/v1/admin/users` - 用户列表（需要 `admin:users:list`）
- `GET /api/v1/admin/recharge-orders` - 充值订单列表
- `GET /api/v1/admin/withdraw-orders` - 提现订单列表
- `POST /api/v1/admin/withdraw-orders/:orderId/audit` - 审核提现订单
- `GET /api/v1/admin/deposit-addresses` - 充值地址列表
- `POST /api/v1/admin/payments/collect` - USDT归集
- `POST /api/v1/admin/payments/batch-collect` - 批量归集

### 权限中间件（Go示例）

```go
// 在路由中使用权限中间件
admin.GET("/users", 
    middleware.AdminAuthMiddleware(),           // 管理员认证
    middleware.RequirePermission(utils.PermissionUsersList),  // 权限检查
    handlers.GetUsers,
)
```

说明：
1. AdminAuthMiddleware - 验证JWT Token，并把管理员信息放入上下文
2. RequirePermission - 检查是否拥有指定权限，无权限返回403

### 安全说明
1. 密码加密：使用 bcrypt
2. Token认证：JWT，包含权限列表
3. 权限控制：接口需对应权限
4. 独立用户表：管理员与前端用户分离
5. 日志记录：记录最后登录时间与IP

### 可扩展方向
- 操作日志、IP白名单、二次验证
- 权限继承、权限缓存（Redis）
- 生产级安全与运维加固

---

## 二、部署与使用指南（原 admin_setup_guide.md）

### 前置条件
1. 已完成数据库迁移（包含RBAC相关表）
2. 数据库服务运行正常
3. Node.js 18+ 已安装（用于Vue前端）

### 快速部署

#### 步骤1：执行数据库迁移
```bash
make migrate
```
将创建管理员、角色、权限及关联表，并初始化：
- 4个默认角色（super_admin, admin, operator, auditor）
- 25个默认权限
- 为 super_admin 角色分配所有权限

#### 步骤2：初始化默认管理员
```bash
make init-admin
```
默认管理员：
- 用户名：`admin`
- 密码：`admin123`

⚠️ 首次登录后请立即修改密码！

#### 步骤3：启动管理后台API服务
```bash
make run-admin
```
服务地址：`http://localhost:8082`

#### 步骤4：安装并启动Vue前端
```bash
cd admin-vue
npm install
npm run dev
```
前端地址：`http://localhost:3000`

### 验证部署

1) 健康检查
```bash
curl http://localhost:8082/health
```
应返回：
```json
{
  "status": "ok",
  "type": "admin-server",
  "port": 8082,
  "time": "..."
}
```

2) 登录接口
```bash
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```
应返回 Token 与管理员信息。

3) 访问前端
打开浏览器访问：`http://localhost:3000`
使用默认账号登录（admin / admin123），并尽快修改密码。

### 配置说明

#### 修改端口
在 `configs/config.yaml`：
```yaml
server:
  admin_port: 8082
```

#### Vue 前端 API 地址
在 `admin-vue/.env`：
```env
VITE_API_BASE_URL=http://localhost:8082
```

### 常见问题（节选）
1. 登录失败：默认管理员未创建或密码错误 → `make init-admin`
2. 403 权限不足：角色或权限未配置/未启用
3. 前端无法连接：服务未启动或端口/API地址错误
4. 数据库表不存在：未执行迁移 → `make migrate`

---

## 三、功能清单与权限（原 admin_features.md）

### 已完成功能

#### 1) 基础
- 管理员登录/登出、JWT认证、权限中间件、CORS

#### 2) 用户管理
- 列表查询（分页/搜索）、详情、更新（昵称/头像/状态）

#### 3) 订单管理
- 充值订单（列表/详情）
- 提现订单（列表/审核）

#### 4) 充值地址
- 列表查询，按用户/链类型筛选

#### 5) USDT归集
- 单用户归集、批量归集、状态跟踪

#### 6) 权限管理（RBAC）
- 角色：增删改、分配权限
- 权限：列表/权限树
- 管理员：增删改、分配角色、改密

#### 7) 操作日志
- 自动记录、列表查询、详情、删除、清理旧日志

#### 8) 系统设置
- 配置管理（增删改）、分组（站点/支付/游戏/系统）
- 类型：string/int/float/bool/json；公开/私有分离；默认配置预置

#### 9) 仪表盘
- 用户/余额/订单/游戏统计与趋势图

### 功能统计（概览）
```
管理后台功能：
├── 用户管理：3个API
├── 订单管理：6个API
├── 充值地址：1个API
├── USDT归集：2个API
├── 权限管理：10个API
├── 操作日志：5个API
├── 系统设置：6个API
└── 仪表盘：2个API
```

### 权限列表示例

#### 仪表盘
- `admin:dashboard:view`

#### 用户
- `admin:users:list`
- `admin:users:detail`
- `admin:users:update`
- `admin:users:ban`

#### 订单
- `admin:recharge_orders:list`
- `admin:recharge_orders:detail`
- `admin:withdraw_orders:list`
- `admin:withdraw_orders:detail`
- `admin:withdraw_orders:audit`

#### 充值地址/支付
- `admin:deposit_addresses:list`
- `admin:payments:collect`
- `admin:payments:batch_collect`

#### 角色/管理员
- `admin:roles:list` / `admin:roles:create` / `admin:roles:update` / `admin:roles:delete` / `admin:roles:assign_permission`
- `admin:admins:list` / `admin:admins:create` / `admin:admins:update` / `admin:admins:delete` / `admin:admins:assign_role`

### 前端页面（完成度）
- 登录、仪表盘、用户管理、充值/提现、充值地址、USDT归集
- 角色、管理员、操作日志、系统设置

### 下一步建议（优先级简表）
- 高：数据导出、消息通知、数据分析
- 中：日志分析、配置版本/历史、系统监控
- 低：文件管理、帮助文档


