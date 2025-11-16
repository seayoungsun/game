# 🚀 服务器部署指南

本文档详细说明如何编译 Go 项目并在服务器上运行。

## 📋 目录

- [环境要求](#环境要求)
- [编译方式](#编译方式)
- [服务器部署](#服务器部署)
- [配置文件](#配置文件)
- [启动服务](#启动服务)
- [进程管理](#进程管理)
- [监控与日志](#监控与日志)
- [常见问题](#常见问题)

---

## 🔧 环境要求

### 服务器环境

- **操作系统**: Linux (推荐 Ubuntu 20.04+ / CentOS 7+)
- **Go 版本**: 1.21+ (仅编译时需要，运行时不需要)
- **MySQL**: 5.7+ 或 8.0+
- **Redis**: 7.0+ (可选，但推荐)
- **系统资源**:
  - CPU: 2核+
  - 内存: 4GB+
  - 磁盘: 20GB+

### 依赖服务

项目需要以下服务：

1. **MySQL** - 数据库
2. **Redis** - 缓存和分布式锁（可选但推荐）
3. **Elasticsearch** - 日志搜索（可选）

---

## 🔨 编译方式

### 方式一：本地编译（推荐）

在本地开发机器上编译，然后上传到服务器。

#### 1. 编译所有服务

```bash
# 在项目根目录执行
make build
```

这会编译三个服务到 `bin/` 目录：
- `bin/api` - API 服务
- `bin/game-server` - 游戏服务器（WebSocket）
- `bin/admin` - 管理后台 API

#### 2. 查看编译结果

```bash
ls -lh bin/
```

输出示例：
```
-rwxr-xr-x  1 user  staff  25M  Jan 15 10:30 admin
-rwxr-xr-x  1 user  staff  28M  Jan 15 10:30 api
-rwxr-xr-x  1 user  staff  30M  Jan 15 10:30 game-server
```

#### 3. 上传到服务器

```bash
# 使用 scp 上传
scp -r bin/ user@your-server:/opt/game-platform/

# 或使用 rsync（推荐，支持断点续传）
rsync -avz --progress bin/ user@your-server:/opt/game-platform/bin/
```

### 方式二：交叉编译（Linux 服务器）

如果服务器是 Linux，可以在本地交叉编译 Linux 版本：

```bash
# 设置编译目标（Linux AMD64）
export GOOS=linux
export GOARCH=amd64

# 编译
make build

# 或手动编译
cd apps/api && GOOS=linux GOARCH=amd64 go build -o ../../bin/api main.go
cd apps/game-server && GOOS=linux GOARCH=amd64 go build -o ../../bin/game-server main.go
cd apps/admin && GOOS=linux GOARCH=amd64 go build -o ../../bin/admin main.go
```

### 方式三：服务器上直接编译

如果服务器已安装 Go 环境：

```bash
# 1. 上传项目代码到服务器
scp -r . user@your-server:/opt/game-platform/

# 2. SSH 登录服务器
ssh user@your-server

# 3. 进入项目目录
cd /opt/game-platform

# 4. 下载依赖
go mod download

# 5. 编译
make build
```

---

## 🖥️ 服务器部署

### 1. 创建部署目录

```bash
# 登录服务器
ssh user@your-server

# 创建项目目录
sudo mkdir -p /opt/game-platform/{bin,configs,logs,scripts}
sudo chown -R $USER:$USER /opt/game-platform
```

### 2. 上传文件

```bash
# 从本地机器执行
# 上传编译好的二进制文件
rsync -avz bin/ user@your-server:/opt/game-platform/bin/

# 上传配置文件
rsync -avz configs/ user@your-server:/opt/game-platform/configs/
```

### 3. 设置执行权限

```bash
# 在服务器上执行
chmod +x /opt/game-platform/bin/*
```

### 4. 创建必要的目录

```bash
mkdir -p /opt/game-platform/logs
```

---

## ⚙️ 配置文件

### 1. 创建生产环境配置

```bash
# 在服务器上
cd /opt/game-platform
cp configs/config.yaml configs/config.prod.yaml
```

### 2. 编辑生产配置

```bash
nano configs/config.prod.yaml
```

**重要配置项：**

```yaml
server:
  mode: release  # 生产环境使用 release
  port: 8080
  game_port: 8081
  admin_port: 8082
  machine_id: 0  # 多实例部署时，每个实例使用不同的 machine_id (0-1023)

database:
  host: localhost  # 或 MySQL 服务器地址
  port: 3306
  user: game_user  # 生产环境使用专用数据库用户
  password: YOUR_STRONG_PASSWORD
  database: game_platform
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: localhost  # 或 Redis 服务器地址
  port: 6379
  password: YOUR_REDIS_PASSWORD
  db: 0

jwt:
  secret: YOUR_STRONG_JWT_SECRET  # 必须修改为强密码
  expiration: 24

log:
  level: info  # 生产环境使用 info 或 warn
  output_path: "/opt/game-platform/logs"
  max_size: 100
  max_backups: 7
  max_age: 30

payment:
  master_mnemonic: "YOUR_MASTER_MNEMONIC"  # 主钱包助记词（必须配置）
  etherscan_api_key: "YOUR_ETHERSCAN_API_KEY"
```

### 3. 使用环境变量（推荐）

生产环境建议使用环境变量覆盖敏感配置：

```bash
# 创建环境变量文件
cat > /opt/game-platform/.env <<EOF
APP_ENV=prod
DATABASE_PASSWORD=your_db_password
REDIS_PASSWORD=your_redis_password
JWT_SECRET=your_jwt_secret
PAYMENT_MASTER_MNEMONIC=your master mnemonic words here
EOF

# 设置权限（仅所有者可读）
chmod 600 /opt/game-platform/.env
```

然后在启动脚本中加载：

```bash
export $(cat /opt/game-platform/.env | xargs)
```

---

## 🚀 启动服务

### 方式一：直接启动（测试用）

```bash
# 启动 API 服务
cd /opt/game-platform
APP_ENV=prod ./bin/api

# 启动游戏服务器（新终端）
APP_ENV=prod ./bin/game-server

# 启动管理后台（新终端）
APP_ENV=prod ./bin/admin
```

### 方式二：后台运行（推荐）

```bash
# 启动 API 服务
nohup APP_ENV=prod ./bin/api > logs/api.log 2>&1 &

# 启动游戏服务器
nohup APP_ENV=prod ./bin/game-server > logs/game-server.log 2>&1 &

# 启动管理后台
nohup APP_ENV=prod ./bin/admin > logs/admin.log 2>&1 &
```

### 方式三：使用 systemd（生产环境推荐）

#### 1. 创建 systemd 服务文件

**API 服务** (`/etc/systemd/system/game-api.service`):

```ini
[Unit]
Description=Game Platform API Service
After=network.target mysql.service redis.service

[Service]
Type=simple
User=game
Group=game
WorkingDirectory=/opt/game-platform
Environment="APP_ENV=prod"
ExecStart=/opt/game-platform/bin/api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-api

# 资源限制
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

**游戏服务器** (`/etc/systemd/system/game-server.service`):

```ini
[Unit]
Description=Game Platform Game Server
After=network.target redis.service

[Service]
Type=simple
User=game
Group=game
WorkingDirectory=/opt/game-platform
Environment="APP_ENV=prod"
ExecStart=/opt/game-platform/bin/game-server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-server

# 资源限制
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

**管理后台** (`/etc/systemd/system/game-admin.service`):

```ini
[Unit]
Description=Game Platform Admin Service
After=network.target mysql.service

[Service]
Type=simple
User=game
Group=game
WorkingDirectory=/opt/game-platform
Environment="APP_ENV=prod"
ExecStart=/opt/game-platform/bin/admin
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-admin

[Install]
WantedBy=multi-user.target
```

#### 2. 创建专用用户（可选但推荐）

```bash
sudo useradd -r -s /bin/false game
sudo chown -R game:game /opt/game-platform
```

#### 3. 启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start game-api
sudo systemctl start game-server
sudo systemctl start game-admin

# 设置开机自启
sudo systemctl enable game-api
sudo systemctl enable game-server
sudo systemctl enable game-admin

# 查看状态
sudo systemctl status game-api
sudo systemctl status game-server
sudo systemctl status game-admin
```

#### 4. 常用命令

```bash
# 查看日志
sudo journalctl -u game-api -f
sudo journalctl -u game-server -f
sudo journalctl -u game-admin -f

# 重启服务
sudo systemctl restart game-api
sudo systemctl restart game-server
sudo systemctl restart game-admin

# 停止服务
sudo systemctl stop game-api
sudo systemctl stop game-server
sudo systemctl stop game-admin
```

---

## 📊 进程管理

### 使用 supervisor（替代方案）

如果不想使用 systemd，可以使用 supervisor：

#### 1. 安装 supervisor

```bash
# Ubuntu/Debian
sudo apt-get install supervisor

# CentOS/RHEL
sudo yum install supervisor
```

#### 2. 创建配置文件

`/etc/supervisor/conf.d/game-platform.conf`:

```ini
[program:game-api]
command=/opt/game-platform/bin/api
directory=/opt/game-platform
user=game
autostart=true
autorestart=true
stderr_logfile=/opt/game-platform/logs/api.err.log
stdout_logfile=/opt/game-platform/logs/api.out.log
environment=APP_ENV=prod

[program:game-server]
command=/opt/game-platform/bin/game-server
directory=/opt/game-platform
user=game
autostart=true
autorestart=true
stderr_logfile=/opt/game-platform/logs/game-server.err.log
stdout_logfile=/opt/game-platform/logs/game-server.out.log
environment=APP_ENV=prod

[program:game-admin]
command=/opt/game-platform/bin/admin
directory=/opt/game-platform
user=game
autostart=true
autorestart=true
stderr_logfile=/opt/game-platform/logs/admin.err.log
stdout_logfile=/opt/game-platform/logs/admin.out.log
environment=APP_ENV=prod
```

#### 3. 启动服务

```bash
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start all
```

---

## 📈 监控与日志

### 1. 查看日志

```bash
# 应用日志
tail -f /opt/game-platform/logs/app.log

# API 服务日志
tail -f /opt/game-platform/logs/api.log

# 游戏服务器日志
tail -f /opt/game-platform/logs/game-server.log

# 如果使用 systemd
sudo journalctl -u game-api -f
```

### 2. 监控端点

项目提供了监控端点（在 API 服务中）：

```bash
# 运行时统计
curl http://localhost:8080/debug/metrics/runtime

# Goroutine 统计
curl http://localhost:8080/debug/metrics/goroutine

# 锁统计
curl http://localhost:8080/debug/metrics/lock

# Worker Pool 统计
curl http://localhost:8080/debug/metrics/worker-pool

# 游戏服务器连接统计
curl http://localhost:8081/stats
```

### 3. 健康检查

```bash
# 检查 API 服务
curl http://localhost:8080/health

# 检查游戏服务器
curl http://localhost:8081/stats
```

---

## 🔒 安全建议

### 1. 防火墙配置

```bash
# 只开放必要端口
sudo ufw allow 8080/tcp  # API 服务
sudo ufw allow 8081/tcp  # 游戏服务器
sudo ufw allow 8082/tcp  # 管理后台（建议仅内网访问）
sudo ufw enable
```

### 2. 使用 Nginx 反向代理（推荐）

```nginx
# /etc/nginx/sites-available/game-platform
upstream api_backend {
    server 127.0.0.1:8080;
}

upstream game_backend {
    server 127.0.0.1:8081;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name ws.yourdomain.com;

    location / {
        proxy_pass http://game_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }
}
```

### 3. 使用 HTTPS

```bash
# 使用 Let's Encrypt 免费证书
sudo apt-get install certbot python3-certbot-nginx
sudo certbot --nginx -d api.yourdomain.com -d ws.yourdomain.com
```

---

## ❓ 常见问题

### 1. 端口被占用

```bash
# 查找占用端口的进程
sudo lsof -i :8080
sudo lsof -i :8081

# 或使用 netstat
sudo netstat -tlnp | grep 8080

# 杀死进程
sudo kill -9 <PID>
```

### 2. 数据库连接失败

```bash
# 检查 MySQL 是否运行
sudo systemctl status mysql

# 检查 MySQL 用户权限
mysql -u root -p
GRANT ALL PRIVILEGES ON game_platform.* TO 'game_user'@'localhost' IDENTIFIED BY 'password';
FLUSH PRIVILEGES;
```

### 3. Redis 连接失败

```bash
# 检查 Redis 是否运行
sudo systemctl status redis

# 测试连接
redis-cli -h localhost -p 6379 -a your_password ping
```

### 4. 文件权限问题

```bash
# 确保二进制文件有执行权限
chmod +x /opt/game-platform/bin/*

# 确保日志目录可写
chmod 755 /opt/game-platform/logs
```

### 5. 内存不足

```bash
# 查看内存使用
free -h

# 查看进程内存使用
ps aux --sort=-%mem | head
```

### 6. 连接数过多

```bash
# 增加文件描述符限制
ulimit -n 65535

# 永久设置（编辑 /etc/security/limits.conf）
* soft nofile 65535
* hard nofile 65535
```

---

## 📝 部署检查清单

- [ ] 编译所有服务 (`make build`)
- [ ] 上传二进制文件到服务器
- [ ] 创建生产环境配置文件
- [ ] 配置数据库连接
- [ ] 配置 Redis 连接（如使用）
- [ ] 设置 JWT Secret
- [ ] 配置主钱包助记词（如使用支付功能）
- [ ] 设置文件权限
- [ ] 配置 systemd 或 supervisor
- [ ] 启动服务并验证
- [ ] 配置防火墙
- [ ] 配置 Nginx 反向代理（可选）
- [ ] 设置日志轮转
- [ ] 配置监控告警（可选）

---

## 🔄 更新部署

### 1. 停止服务

```bash
sudo systemctl stop game-api game-server game-admin
```

### 2. 备份当前版本

```bash
cp -r /opt/game-platform/bin /opt/game-platform/bin.backup.$(date +%Y%m%d)
```

### 3. 上传新版本

```bash
rsync -avz bin/ user@your-server:/opt/game-platform/bin/
```

### 4. 重启服务

```bash
sudo systemctl start game-api game-server game-admin
```

### 5. 验证

```bash
# 检查服务状态
sudo systemctl status game-api game-server game-admin

# 检查日志
sudo journalctl -u game-api -n 50
```

---

## 📚 相关文档

- [README.md](../README.md) - 项目总览
- [配置说明](../README.md#环境配置) - 配置文件详解
- [API 文档](./api_summary.md) - API 接口文档
- [监控指南](./monitoring_guide.md) - 监控系统使用

---

**最后更新**: 2025-01-15

