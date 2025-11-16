#!/bin/bash

# 游戏平台部署脚本
# 使用方法: ./scripts/deploy.sh [选项]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
DEPLOY_DIR="/opt/game-platform"
SERVICE_USER="game"
BUILD_DIR="./bin"
ENV_FILE=".env"

# 显示帮助信息
show_help() {
    cat << EOF
游戏平台部署脚本

使用方法:
    $0 [选项]

选项:
    -h, --help              显示帮助信息
    -d, --dir DIR           部署目录 (默认: $DEPLOY_DIR)
    -u, --user USER         服务运行用户 (默认: $SERVICE_USER)
    -b, --build             编译项目
    -s, --server SERVER     服务器地址 (例如: user@192.168.1.100)
    -c, --config            仅上传配置文件
    -e, --env ENV           环境名称 (默认: prod)
    --skip-build            跳过编译步骤
    --skip-upload           跳过上传步骤
    --setup-systemd         设置 systemd 服务

示例:
    # 本地编译
    $0 --build

    # 编译并上传到服务器
    $0 --build --server user@192.168.1.100

    # 仅上传配置文件
    $0 --config --server user@192.168.1.100

    # 设置 systemd 服务
    $0 --setup-systemd --server user@192.168.1.100

EOF
}

# 打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        error "$1 未安装，请先安装 $1"
    fi
}

# 编译项目
build_project() {
    info "开始编译项目..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "Go 未安装，请先安装 Go 1.21+"
    fi
    
    # 创建 bin 目录
    mkdir -p $BUILD_DIR
    
    # 编译
    info "编译 API 服务..."
    cd apps/api && go build -o ../../$BUILD_DIR/api main.go && cd ../..
    
    info "编译游戏服务器..."
    cd apps/game-server && go build -o ../../$BUILD_DIR/game-server main.go && cd ../..
    
    info "编译管理后台..."
    cd apps/admin && go build -o ../../$BUILD_DIR/admin main.go && cd ../..
    
    # 检查编译结果
    if [ ! -f "$BUILD_DIR/api" ] || [ ! -f "$BUILD_DIR/game-server" ] || [ ! -f "$BUILD_DIR/admin" ]; then
        error "编译失败，请检查错误信息"
    fi
    
    info "编译完成！"
    ls -lh $BUILD_DIR/
}

# 上传文件到服务器
upload_to_server() {
    if [ -z "$SERVER" ]; then
        error "请指定服务器地址 (使用 -s 或 --server)"
    fi
    
    info "上传文件到服务器: $SERVER"
    
    # 检查 rsync 或 scp
    if command -v rsync &> /dev/null; then
        UPLOAD_CMD="rsync"
    elif command -v scp &> /dev/null; then
        UPLOAD_CMD="scp"
    else
        error "未找到 rsync 或 scp，请先安装"
    fi
    
    # 创建远程目录
    ssh $SERVER "sudo mkdir -p $DEPLOY_DIR/{bin,configs,logs} && sudo chown -R \$USER:\$USER $DEPLOY_DIR"
    
    if [ "$UPLOAD_CMD" = "rsync" ]; then
        # 上传二进制文件
        if [ "$SKIP_BUILD" != "true" ] && [ -d "$BUILD_DIR" ]; then
            info "上传二进制文件..."
            rsync -avz --progress $BUILD_DIR/ $SERVER:$DEPLOY_DIR/bin/
        fi
        
        # 上传配置文件
        if [ -d "configs" ]; then
            info "上传配置文件..."
            rsync -avz --progress configs/ $SERVER:$DEPLOY_DIR/configs/
        fi
    else
        # 使用 scp
        if [ "$SKIP_BUILD" != "true" ] && [ -d "$BUILD_DIR" ]; then
            info "上传二进制文件..."
            scp -r $BUILD_DIR/* $SERVER:$DEPLOY_DIR/bin/
        fi
        
        if [ -d "configs" ]; then
            info "上传配置文件..."
            scp -r configs/* $SERVER:$DEPLOY_DIR/configs/
        fi
    fi
    
    # 设置执行权限
    ssh $SERVER "chmod +x $DEPLOY_DIR/bin/*"
    
    info "上传完成！"
}

# 仅上传配置文件
upload_config_only() {
    if [ -z "$SERVER" ]; then
        error "请指定服务器地址 (使用 -s 或 --server)"
    fi
    
    info "上传配置文件到服务器: $SERVER"
    
    ssh $SERVER "mkdir -p $DEPLOY_DIR/configs"
    
    if command -v rsync &> /dev/null; then
        rsync -avz --progress configs/ $SERVER:$DEPLOY_DIR/configs/
    else
        scp -r configs/* $SERVER:$DEPLOY_DIR/configs/
    fi
    
    info "配置文件上传完成！"
}

# 设置 systemd 服务
setup_systemd() {
    if [ -z "$SERVER" ]; then
        error "请指定服务器地址 (使用 -s 或 --server)"
    fi
    
    info "设置 systemd 服务..."
    
    # 创建服务文件内容
    cat > /tmp/game-api.service << EOF
[Unit]
Description=Game Platform API Service
After=network.target mysql.service redis.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR
Environment="APP_ENV=${APP_ENV:-prod}"
ExecStart=$DEPLOY_DIR/bin/api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-api
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

    cat > /tmp/game-server.service << EOF
[Unit]
Description=Game Platform Game Server
After=network.target redis.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR
Environment="APP_ENV=${APP_ENV:-prod}"
ExecStart=$DEPLOY_DIR/bin/game-server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-server
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

    cat > /tmp/game-admin.service << EOF
[Unit]
Description=Game Platform Admin Service
After=network.target mysql.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR
Environment="APP_ENV=${APP_ENV:-prod}"
ExecStart=$DEPLOY_DIR/bin/admin
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=game-admin

[Install]
WantedBy=multi-user.target
EOF

    # 上传服务文件
    scp /tmp/game-*.service $SERVER:/tmp/
    
    # 在服务器上安装服务
    ssh $SERVER << 'ENDSSH'
        sudo mv /tmp/game-*.service /etc/systemd/system/
        sudo systemctl daemon-reload
        
        # 创建服务用户（如果不存在）
        if ! id "$SERVICE_USER" &>/dev/null; then
            sudo useradd -r -s /bin/false $SERVICE_USER
        fi
        
        # 设置目录权限
        sudo chown -R $SERVICE_USER:$SERVICE_USER $DEPLOY_DIR
        
        echo "Systemd 服务已安装！"
        echo "使用以下命令启动服务:"
        echo "  sudo systemctl start game-api"
        echo "  sudo systemctl start game-server"
        echo "  sudo systemctl start game-admin"
        echo ""
        echo "设置开机自启:"
        echo "  sudo systemctl enable game-api"
        echo "  sudo systemctl enable game-server"
        echo "  sudo systemctl enable game-admin"
ENDSSH
    
    # 清理临时文件
    rm -f /tmp/game-*.service
    
    info "Systemd 服务设置完成！"
}

# 解析命令行参数
BUILD=false
CONFIG_ONLY=false
SETUP_SYSTEMD=false
SKIP_BUILD=false
SKIP_UPLOAD=false
APP_ENV="prod"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -d|--dir)
            DEPLOY_DIR="$2"
            shift 2
            ;;
        -u|--user)
            SERVICE_USER="$2"
            shift 2
            ;;
        -b|--build)
            BUILD=true
            shift
            ;;
        -s|--server)
            SERVER="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_ONLY=true
            shift
            ;;
        -e|--env)
            APP_ENV="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-upload)
            SKIP_UPLOAD=true
            shift
            ;;
        --setup-systemd)
            SETUP_SYSTEMD=true
            shift
            ;;
        *)
            error "未知选项: $1"
            ;;
    esac
done

# 主流程
main() {
    info "游戏平台部署脚本"
    info "=================="
    
    # 编译
    if [ "$BUILD" = "true" ] && [ "$SKIP_BUILD" != "true" ]; then
        build_project
    fi
    
    # 仅上传配置
    if [ "$CONFIG_ONLY" = "true" ]; then
        upload_config_only
        exit 0
    fi
    
    # 上传文件
    if [ "$SKIP_UPLOAD" != "true" ] && [ -n "$SERVER" ]; then
        upload_to_server
    fi
    
    # 设置 systemd
    if [ "$SETUP_SYSTEMD" = "true" ]; then
        setup_systemd
    fi
    
    info "部署完成！"
    
    if [ -n "$SERVER" ]; then
        echo ""
        info "下一步操作:"
        echo "  1. SSH 登录服务器: ssh $SERVER"
        echo "  2. 编辑配置文件: nano $DEPLOY_DIR/configs/config.prod.yaml"
        echo "  3. 启动服务:"
        echo "     - 直接运行: cd $DEPLOY_DIR && APP_ENV=prod ./bin/api"
        echo "     - 或使用 systemd: sudo systemctl start game-api"
    fi
}

# 执行主流程
main

