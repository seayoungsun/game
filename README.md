# 🎮 游戏平台项目

一个在线棋牌游戏平台（后端 Go + WebSocket 实时通信 + 简易 Web 前端 + 管理后台）。当前聚焦用户认证、房间/游戏流程与基础管理后台。

## 📋 目录

- [快速开始](#快速开始)
- [环境配置](#环境配置)
- [数据库初始化](#数据库初始化)
- [API 概览](#api-概览)
- [构建与运行](#构建与运行)
- [项目结构](#项目结构)
- [Docker 部署](#docker-部署)
- [技术文档](#技术文档)
- [贡献与许可证](#贡献与许可证)

---

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone <repository-url>
cd game
```

### 2. 配置环境

```bash
cp configs/config.local.yaml.example configs/config.local.yaml
# 根据本地环境修改数据库、Redis 等配置
```

### 3. 初始化数据库（首选迁移）

```bash
# 创建数据库（如未创建）
mysql -u root -p -e 'CREATE DATABASE game_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;'

# 执行迁移
make migrate
```

如需快速样例数据，可选导入 `database.sql`（以迁移为准，避免冲突）。

### 4. 启动服务

```bash
# 终端1：API 服务（端口 8080，负责 HTTP 与静态站点 web/）
make run-api

# 终端2：游戏服务器（端口 8081，WebSocket）
make run-game

# 终端3（可选）：管理后台 API（端口 8082）
make run-admin
```

### 5. 访问
- 用户站点（静态）：`http://localhost:8080`（由 API 映射 `web/` 目录）
- 管理后台前端（独立工程，可选）：`admin-vue/`
  ```bash
  cd admin-vue
  npm install
  npm run dev   # 默认 http://localhost:3000
  ```

---

## ⚙️ 环境配置

### 版本要求
- Go 1.21+
- MySQL 5.7+/8.0+
- Redis 7.0+
- Node.js 18+（仅管理后台前端）

### 配置加载顺序
1. 内置默认值：`internal/config/config.go`
2. 基础配置：`configs/config.yaml`
3. 环境配置：`configs/config.<env>.yaml`（`APP_ENV` 控制，默认 `local`）
4. 环境变量覆盖（如 `REDIS_HOST`）

常见启动示例：
```bash
APP_ENV=local make run-api
APP_ENV=prod DATABASE_PASSWORD=*** ./bin/api
```

---

## 🗄️ 数据库初始化

优先使用迁移：
```bash
make migrate
```

可选（不推荐为主流程）：导入 `database.sql` 以获得示例数据。若与迁移冲突，请以迁移为准。

---

## 📡 API 概览

- 基础 URL：`http://localhost:8080/api/v1`
- 认证方式：JWT（`Authorization: Bearer <token>`）
- 主要能力：用户注册/登录/资料、房间创建/加入/离开/列表、准备/开始/出牌、游戏状态查询
- 实时通信：`ws://localhost:8081/ws?token=<token>`，用于房间广播和状态推送

更详细的高层说明见：`docs/api_overview.md`

---

## 🔧 构建与运行

常用命令（节选）：
```bash
# 运行
make run-api           # API 服务 :8080
make run-game          # 游戏服务器 :8081
make run-admin         # 管理后台 API :8082

# 迁移与工具
make migrate
make fmt
make vet
make test

# 构建（当前平台）
make build

# 交叉编译
make build-linux
make build-linux-arm64
make build-windows
make build-darwin
make build-darwin-arm64
make build-all

# 清理
make clean
```

说明：
- Makefile 中的前端大厅 `client-lobby` 相关命令已不再使用，前端请使用 `admin-vue` 独立工程。
- 产物默认输出至 `bin/`；建议将 `bin/`、日志与 `node_modules/` 加入 `.gitignore`（已配置）。

---

## 🧭 项目结构

```
game/
├── apps/
│   ├── api/              # 用户侧 API（HTTP + 静态 web/ 映射）
│   ├── game-server/      # 游戏服务器（WebSocket）
│   └── admin/            # 管理后台 API
├── admin-vue/            # 管理后台前端（独立 Vue3 + Vite）
├── internal/             # 内部库（配置/存储/服务编排等）
│   ├── bootstrap/
│   ├── cache/
│   ├── config/
│   ├── database/
│   ├── elasticsearch/
│   ├── logger/
│   ├── metrics/
│   ├── middleware/
│   ├── repository/
│   ├── service/
│   └── storage/
├── pkg/                  # 共享模型与服务
│   ├── models/
│   ├── services/
│   └── utils/
├── web/                  # 静态站点（由 API 服务映射）
├── configs/              # 配置文件
├── migrations/           # 数据库迁移
├── scripts/              # 工具脚本（部署、测试、迁移）
├── docker/               # docker-compose 与相关配置
├── docs/                 # 项目文档
└── bin/                  # 构建产物（已在 .gitignore）
```

---

## 🐳 Docker 部署

```bash
cd docker
docker-compose up -d
```

将启动：
- MySQL
- Redis
- Elasticsearch + Kibana

停止：
```bash
docker-compose down
```

更多部署细节见：`docs/deployment_guide.md`

---

## 📚 技术文档

- API 概览：`docs/api_overview.md`
- 管理后台综合：`docs/admin_guide.md`
- 数据库设计：`docs/database.md`
- 部署指南：`docs/deployment_guide.md`
- 其他专题：`docs/` 目录

（已清理过时文档引用，不再包含详尽的接口逐条示例。）

---

## 🤝 贡献与许可证

欢迎提交 Issue / PR 改进项目。

许可证：MIT License

