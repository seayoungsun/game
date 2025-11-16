#!/bin/bash

# 本地开发环境启动脚本

echo "🚀 启动本地开发环境..."

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go 1.21+"
    echo "安装命令: brew install go"
    exit 1
fi

echo "✅ Go版本: $(go version)"

# 检查配置文件
if [ ! -f "configs/config.local.yaml" ]; then
    echo "📝 配置文件不存在，正在创建..."
    cp configs/config.local.yaml.example configs/config.local.yaml 2>/dev/null || cp configs/config.local.yaml configs/config.local.yaml.bak
    echo "⚠️  请编辑 configs/config.local.yaml 配置数据库信息"
    echo "   然后重新运行此脚本"
    exit 1
fi

# 下载依赖
echo "📦 下载Go依赖..."
go mod download

# 创建日志目录
mkdir -p logs

# 检查MySQL连接（可选）
echo "🔍 检查MySQL连接..."
if command -v mysql &> /dev/null; then
    echo "✅ MySQL客户端已安装"
else
    echo "⚠️  MySQL客户端未安装（不影响开发）"
fi

# 检查Redis（可选）
echo "🔍 检查Redis..."
if command -v redis-cli &> /dev/null; then
    redis-cli ping &> /dev/null
    if [ $? -eq 0 ]; then
        echo "✅ Redis已启动"
    else
        echo "⚠️  Redis未启动（可选，某些功能可能不可用）"
        echo "   启动命令: brew services start redis"
    fi
else
    echo "⚠️  Redis未安装（可选）"
    echo "   安装命令: brew install redis"
fi

echo ""
echo "🎉 环境检查完成！"
echo ""
echo "下一步："
echo "1. 启动API服务: make run-api"
echo "2. 启动游戏服务器: make run-game"
echo "3. 测试服务: curl http://localhost:8080/health"
echo ""










