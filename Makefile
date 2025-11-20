.PHONY: help build build-linux build-windows build-darwin build-all run-api run-game run-admin docker-up docker-down migrate test clean local-setup init-admin

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译项目（当前平台）
	@echo "编译API服务..."
	cd apps/api && go build -o ../../bin/api .
	@echo "编译游戏服务器..."
	cd apps/game-server && go build -o ../../bin/game-server .
	@echo "编译管理后台..."
	cd apps/admin && go build -o ../../bin/admin .

build-linux: ## 交叉编译 Linux 版本（用于服务器）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 交叉编译 Linux 版本 (amd64)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@mkdir -p bin
	@echo "编译API服务..."
	cd apps/api && GOOS=linux GOARCH=amd64 go build -o ../../bin/api-linux .
	@echo "编译游戏服务器..."
	cd apps/game-server && GOOS=linux GOARCH=amd64 go build -o ../../bin/game-server-linux .
	@echo "编译管理后台..."
	cd apps/admin && GOOS=linux GOARCH=amd64 go build -o ../../bin/admin-linux .
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 编译完成！文件在 bin/ 目录："
	@ls -lh bin/*-linux 2>/dev/null || true

build-linux-arm64: ## 交叉编译 Linux ARM64 版本
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 交叉编译 Linux 版本 (arm64)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@mkdir -p bin
	@echo "编译API服务..."
	cd apps/api && GOOS=linux GOARCH=arm64 go build -o ../../bin/api-linux-arm64 .
	@echo "编译游戏服务器..."
	cd apps/game-server && GOOS=linux GOARCH=arm64 go build -o ../../bin/game-server-linux-arm64 .
	@echo "编译管理后台..."
	cd apps/admin && GOOS=linux GOARCH=arm64 go build -o ../../bin/admin-linux-arm64 .
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 编译完成！文件在 bin/ 目录："
	@ls -lh bin/*-linux-arm64 2>/dev/null || true

build-windows: ## 交叉编译 Windows 版本
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 交叉编译 Windows 版本 (amd64)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@mkdir -p bin
	@echo "编译API服务..."
	cd apps/api && GOOS=windows GOARCH=amd64 go build -o ../../bin/api-windows.exe .
	@echo "编译游戏服务器..."
	cd apps/game-server && GOOS=windows GOARCH=amd64 go build -o ../../bin/game-server-windows.exe $$(find . -name "*.go" -not -path "./web/*" | tr '\n' ' ')
	@echo "编译管理后台..."
	cd apps/admin && GOOS=windows GOARCH=amd64 go build -o ../../bin/admin-windows.exe .
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 编译完成！文件在 bin/ 目录："
	@ls -lh bin/*-windows.exe 2>/dev/null || true

build-darwin: ## 交叉编译 macOS 版本
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 交叉编译 macOS 版本 (amd64)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@mkdir -p bin
	@echo "编译API服务..."
	cd apps/api && GOOS=darwin GOARCH=amd64 go build -o ../../bin/api-darwin-amd64 .
	@echo "编译游戏服务器..."
	cd apps/game-server && GOOS=darwin GOARCH=amd64 go build -o ../../bin/game-server-darwin-amd64 $$(find . -name "*.go" -not -path "./web/*" | tr '\n' ' ')
	@echo "编译管理后台..."
	cd apps/admin && GOOS=darwin GOARCH=amd64 go build -o ../../bin/admin-darwin-amd64 .
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 编译完成！文件在 bin/ 目录："
	@ls -lh bin/*-darwin-amd64 2>/dev/null || true

build-darwin-arm64: ## 交叉编译 macOS ARM64 版本（Apple Silicon）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 交叉编译 macOS 版本 (arm64)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@mkdir -p bin
	@echo "编译API服务..."
	cd apps/api && GOOS=darwin GOARCH=arm64 go build -o ../../bin/api-darwin-arm64 .
	@echo "编译游戏服务器..."
	cd apps/game-server && GOOS=darwin GOARCH=arm64 go build -o ../../bin/game-server-darwin-arm64 $$(find . -name "*.go" -not -path "./web/*" | tr '\n' ' ')
	@echo "编译管理后台..."
	cd apps/admin && GOOS=darwin GOARCH=arm64 go build -o ../../bin/admin-darwin-arm64 .
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 编译完成！文件在 bin/ 目录："
	@ls -lh bin/*-darwin-arm64 2>/dev/null || true

build-all: ## 编译所有平台版本
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔨 编译所有平台版本"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(MAKE) build-linux
	@$(MAKE) build-windows
	@$(MAKE) build-darwin
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ 所有平台编译完成！"
	@ls -lh bin/ 2>/dev/null || true

run-api: ## 运行API服务（端口8080）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🚀 启动 API 服务 (端口: 8080)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@cd apps/api && go run .

run-game: ## 运行游戏服务器（端口8081）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🎮 启动游戏服务器 (端口: 8081)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@cd apps/game-server && go run .

run-admin: ## 运行管理后台服务（端口8082）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔐 启动管理后台服务 (端口: 8082)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@cd apps/admin && go run .

local-setup: ## 本地环境初始化检查
	@bash scripts/local-start.sh

docker-up: ## 启动Docker服务（MySQL + Redis + ES）
	docker-compose -f docker/docker-compose.yml up -d
	@echo "等待服务启动..."
	sleep 5
	@echo "服务已启动，访问:"
	@echo "  - MySQL: localhost:3306"
	@echo "  - Redis: localhost:6379"
	@echo "  - Elasticsearch: http://localhost:9200"
	@echo "  - Kibana: http://localhost:5601"

docker-down: ## 停止Docker服务
	docker-compose -f docker/docker-compose.yml down

docker-logs: ## 查看Docker日志
	docker-compose -f docker/docker-compose.yml logs -f

migrate: ## 执行数据库迁移
	cd scripts/migrate && go run main.go

init-admin: ## 初始化默认管理员（执行迁移后运行）
	@echo "初始化默认管理员..."
	@cd scripts && go run init_admin.go

test: ## 运行测试
	go test ./... -v

clean: ## 清理编译文件
	rm -rf bin/
	rm -rf logs/
	go clean

install-deps: ## 安装依赖
	go mod download
	go mod tidy

fmt: ## 格式化代码
	go fmt ./...
	gofmt -w .

vet: ## 代码检查
	go vet ./...

test-api: ## 测试所有API接口
	@bash scripts/test_api.sh

run-lobby: ## 运行前端大厅（端口3000）
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🎨 启动 Vue 大厅 (端口: 3000)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@cd client-lobby && npm run dev

install-lobby: ## 安装前端依赖
	@echo "安装 Vue 大厅依赖..."
	@cd client-lobby && npm install

build-lobby: ## 构建前端大厅
	@echo "构建 Vue 大厅..."
	@cd client-lobby && npm run build
