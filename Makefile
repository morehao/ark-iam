# Go应用程序Makefile

# ============================================================
# 构建相关变量
# ============================================================
APP        =
BINARY     = $(APP)
MAIN_DIR   = ./apps/$(APP)/cmd
BUILD_DIR  = ./backend/output/build
VERSION    = $(shell date +%Y%m%d%H%M%S)-$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

APP_CONFIG_PATH = /app/config.yaml

# go命令的环境变量
GO_ENV = CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct

# ============================================================
# Docker 相关变量
# ============================================================
# 获取 git tag（如果存在），不存在时为空字符串
GIT_TAG    = $(shell git describe --tags --exact-match 2>/dev/null || echo "")
# 获取 commit hash 短格式
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# 构建镜像 tag：如果有 git tag 则使用 tag-commit，否则使用 commit
DOCKER_TAG        = $(if $(GIT_TAG),$(GIT_TAG)-$(GIT_COMMIT),$(GIT_COMMIT))
# 完整的镜像名称：appname:tag
DOCKER_IMAGE      = $(APP):$(DOCKER_TAG)
# 容器对外暴露端口（可通过命令行覆盖，例如：make docker-run APP=foo PORT=9090）
PORT              = 8099

# ============================================================
# 伪目标
# ============================================================
.PHONY: all build build-env clean run lint test swag codegen \
        docker-build docker-run check-image \
        list-apps deps tidy update-dep dev-frontend stop-frontend e2e e2e-sso e2e-test help

# ============================================================
# 通用入口：清理、依赖、构建并运行
# ============================================================
all: clean deps build run

# ============================================================
# 工具函数
# ============================================================

# 验证 APP 参数是否有效，同时列出可用应用供参考
define validate_app
	@if [ -z "$(APP)" ]; then \
		echo "❌ 请使用 APP=<名称> 指定要操作的应用程序，例如：make build APP=demoapp"; \
		echo "   可用的应用程序：$$(ls ./backend/apps 2>/dev/null | tr '\n' ' ')"; \
		exit 1; \
	fi
	@if [ ! -d "./backend/apps/$(APP)" ]; then \
		echo "❌ 应用程序 '$(APP)' 不存在于 ./backend/apps 目录下"; \
		echo "   可用的应用程序：$$(ls ./backend/apps 2>/dev/null | tr '\n' ' ')"; \
		exit 1; \
	fi
endef

# ============================================================
# 构建
# ============================================================

# 构建应用程序（本机环境）
build:
	$(call validate_app)
	@echo "🔨 正在构建应用程序 $(APP)..."
	@mkdir -p $(BUILD_DIR)
	@cd backend && go work sync && cd ..
	@cd backend && go build -ldflags="-X 'main.BuildVersion=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY) $(MAIN_DIR)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(BINARY)"

# 为指定环境构建（默认注入 CGO_ENABLED=0，适合容器/Linux 交叉编译）
build-env:
	$(call validate_app)
	@echo "🔨 正在为 [$(GO_ENV)] 构建 $(APP)..."
	@mkdir -p $(BUILD_DIR)
	@cd backend && go work sync && cd ..
	@cd backend && $(GO_ENV) go build -ldflags="-X 'main.BuildVersion=$(VERSION)'" -o $(BUILD_DIR)/$(BINARY) $(MAIN_DIR)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(BINARY)"

# ============================================================
# 清理
# ============================================================
clean:
	@echo "🧹 正在清理构建目录..."
	@rm -rf $(BUILD_DIR)
	@echo "✅ 已清理构建目录"

# ============================================================
# 运行 & 测试
# ============================================================

# 直接用 go run 启动（开发调试用）
run:
	$(call validate_app)
	@echo "🚀 正在运行应用程序 $(APP)..."
	@cd backend && go work sync && cd ..
	@cd backend && go run $(MAIN_DIR)

# 运行单元测试
test:
	$(call validate_app)
	@echo "🧪 正在运行测试..."
	@cd backend && go work sync && cd ..
	@cd backend && go test ./apps/$(APP)/... -v

# ============================================================
# 依赖管理
# ============================================================

# 同步 workspace 并更新所有子模块依赖
deps:
	@echo "📦 正在下载依赖项..."
	@cd backend && go work sync && cd ..
	@find ./backend -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		echo "  => $$dir"; \
		(cd $$dir && $(GO_ENV) go mod download && go mod tidy); \
	done
	@echo "✅ 依赖项已更新"

# 仅执行 go mod tidy（不下载全量依赖，速度更快）
tidy:
	@find ./backend -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		echo "==> Tidy in $$dir"; \
		(cd $$dir && go mod tidy); \
	done
	@cd backend && go work sync && cd ..
	@echo "✅ go mod tidy 完成"

# 升级指定依赖包（仅在引用该包的模块中执行）
# 用法：make update-dep ARGS="-u github.com/morehao/golib@v0.1.1"
update-dep:
	@if [ -z "$(ARGS)" ]; then \
		echo "❌ 请提供 ARGS 参数"; \
		echo "   用法：make update-dep ARGS=\"-u github.com/morehao/golib@v0.1.1\""; \
		exit 1; \
	fi
	@find ./backend -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		pkg=$$(echo "$(ARGS)" | grep -oE '[a-zA-Z0-9._/-]+/[a-zA-Z0-9._/-]+' | head -1); \
		if grep -q "$$pkg" "$$modfile"; then \
			echo "==> Updating in $$dir"; \
			(cd $$dir && go get $(ARGS) && go mod tidy); \
		fi \
	done
	@$(MAKE) tidy

# ============================================================
# 代码生成
# ============================================================

# 生成 Swagger 文档
swag:
	$(call validate_app)
	@echo "📚 正在生成 Swagger 文档..."
	@which swag > /dev/null 2>&1 || (echo "⚠️  swag 未安装，正在安装..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@cd backend && swag init \
		--parseDependency \
		--parseInternal \
		-g app.go \
		--dir apps/$(APP) \
		--output apps/$(APP)/docs \
		--outputTypes go \
		--instanceName $(APP)
	@echo "✅ Swagger 文档已生成：backend/apps/$(APP)/docs"

CLI_VERSION := v0.1.12
CLI_PKG     := github.com/morehao/gocli

# 代码生成（API / module / model 等）
# 用法：make codegen APP=demoapp COMMAND=api
codegen:
	$(call validate_app)
	@if [ -z "$(COMMAND)" ]; then \
		echo "❌ 请使用 COMMAND 参数指定生成命令，例如：make codegen APP=$(APP) COMMAND=api"; \
		echo "   支持的命令：api, module, model"; \
		exit 1; \
		fi
	@if ! command -v gocli >/dev/null 2>&1; then \
		echo "⚠️  未检测到 gocli，正在安装 $(CLI_VERSION)..."; \
		go install $(CLI_PKG)@$(CLI_VERSION); \
	else \
		INSTALLED_VER=$$(go version -m $$(which gocli) 2>/dev/null \
			| grep -E "^\s+mod\s+$(CLI_PKG)" | awk '{print $$3}' || echo ""); \
		echo "🔍 已安装的 gocli 版本: $$INSTALLED_VER，目标版本: $(CLI_VERSION)"; \
		if [ "$$INSTALLED_VER" != "$(CLI_VERSION)" ]; then \
			echo "⚠️  版本不匹配，重新安装 $(CLI_VERSION)..."; \
			go install $(CLI_PKG)@$(CLI_VERSION); \
		else \
			echo "✅ gocli 版本已是最新"; \
		fi; \
	fi
	@echo "🔧 开始生成代码：APP=$(APP)，COMMAND=$(COMMAND)"
	@cd backend && gocli generate $(COMMAND) -a $(APP)

# ============================================================
# Docker
# ============================================================

# 构建 Docker 镜像
docker-build:
	$(call validate_app)
	@echo "🐳 正在构建 $(APP) 的 Docker 镜像..."
	@echo "   镜像名称  : $(DOCKER_IMAGE)"
	@echo "   Git Tag   : $(if $(GIT_TAG),$(GIT_TAG),无)"
	@echo "   Git Commit: $(GIT_COMMIT)"
	@docker buildx build -t $(DOCKER_IMAGE) -f ./backend/apps/$(APP)/scripts/Dockerfile backend
	@echo "✅ Docker 镜像 $(DOCKER_IMAGE) 构建完成"

# 运行 Docker 容器（镜像不存在时自动构建）
# 用法：make docker-run APP=demoapp [PORT=9090]
docker-run: check-image
	@echo "🚀 正在运行 $(APP) 容器..."
	@echo "   使用镜像 : $(DOCKER_IMAGE)"
	@echo "   端口映射 : $(PORT):$(PORT)"
	-@docker rm -f $(APP) 2>/dev/null || true
	@docker run -d \
		--name $(APP) \
		-e APP_CONFIG_PATH=$(APP_CONFIG_PATH) \
		-p $(PORT):$(PORT) \
		$(DOCKER_IMAGE)
	@echo "✅ 容器 $(APP) 已启动，服务地址：http://localhost:$(PORT)"

# 检查镜像是否存在，不存在才构建（保留缓存）
check-image:
	$(call validate_app)
	@if [ -z "$$(docker images -q $(DOCKER_IMAGE) 2>/dev/null)" ]; then \
		echo "📦 镜像 $(DOCKER_IMAGE) 不存在，开始构建..."; \
		$(MAKE) docker-build APP=$(APP); \
	else \
		echo "✅ 镜像 $(DOCKER_IMAGE) 已存在，跳过构建"; \
	fi

# ============================================================
# 前端开发
# ============================================================

# 并行启动所有前端服务（开发调试用）
dev-frontend:
	@echo "🚀 正在并行启动所有前端服务..."
	@cd frontend && pnpm run dev:all

# 停止所有前端服务（本地测试用）
stop-frontend:
	@echo "🛑 正在停止前端服务..."
	@pkill -f "vite.*ark-iam" 2>/dev/null && echo "✅ 已停止" || echo "⚠️  没有运行中的前端服务"

# 运行 E2E 测试（需先安装依赖：cd e2e && npm install && npx playwright install chromium）
.PHONY: e2e
e2e:
	@echo "🧪 运行全部 E2E 测试..."
	@cd e2e && npx playwright test

# 运行 OIDC SSO E2E 测试（含 SSO 自动登录、登出后 session 清除等）
.PHONY: e2e-sso
e2e-sso:
	@echo "🧪 运行 OIDC SSO E2E 测试..."
	@cd e2e && npx playwright test oidc-sso.spec.ts

# 按名称匹配运行 E2E 测试
# 用法：make e2e-test TEST="管理平台登出"
.PHONY: e2e-test
e2e-test:
	@if [ -z "$(TEST)" ]; then \
		echo "❌ 请使用 TEST 参数指定测试名称"; \
		echo "   用法：make e2e-test TEST=\"管理平台登出\""; \
		exit 1; \
	fi
	@echo "🧪 运行 E2E 测试: $(TEST)"
	@cd e2e && npx playwright test --grep "$(TEST)"

# ============================================================
# 其他工具
# ============================================================

# 列出所有可用的应用程序
list-apps:
	@echo "📂 可用的应用程序:"
	@ls -1 ./backend/apps

# 运行代码检查工具（在 workspace 各模块目录内分别执行）
lint:
	@echo "🔍 正在运行代码检查工具..."
	@cd backend && go work edit -json | grep -o '"DiskPath": "[^"]*"' | awk '{gsub(/"/, "", $$2); print $$2}' | while read mod; do \
		echo "  => lint $$mod"; \
		(cd $$mod && golangci-lint run ./...); \
	done

# ============================================================
# 帮助信息
# ============================================================
help:
	@echo ""
	@echo "🆘 可用命令："
	@echo ""
	@echo "  构建 & 运行"
	@echo "    make build     APP=<名称>              本机环境构建"
	@echo "    make build-env APP=<名称>              指定环境变量构建（适合交叉编译）"
	@echo "    make run       APP=<名称>              go run 直接启动（开发调试）"
	@echo "    make clean                             清理构建产物"
	@echo "    make all       APP=<名称>              clean → deps → build → run"
	@echo ""
	@echo "  测试 & 检查"
	@echo "    make test      APP=<名称>              运行单元测试"
	@echo "    make lint                              运行 golangci-lint"
	@echo "    make e2e                                运行全部 E2E 测试"
	@echo "    make e2e-sso                            运行 OIDC SSO E2E 测试"
	@echo "    make e2e-test TEST=\"关键词\"           按名称匹配运行 E2E 测试"
	@echo ""
	@echo "  依赖管理"
	@echo "    make deps                              同步 workspace 并更新全部模块依赖"
	@echo "    make tidy                              对所有模块执行 go mod tidy"
	@echo "    make update-dep ARGS=\"-u <pkg@ver>\"   升级指定依赖包"
	@echo ""
	@echo "  代码生成"
	@echo "    make swag      APP=<名称>              生成 Swagger 文档"
	@echo "    make codegen   APP=<名称> COMMAND=<api|module|model>  生成代码"
	@echo ""
	@echo "  Docker"
	@echo "    make docker-build APP=<名称>           构建 Docker 镜像"
	@echo "    make docker-run   APP=<名称> [PORT=N]  运行容器（镜像不存在时自动构建）"
	@echo ""
	@echo "  前端"
	@echo "    make dev-frontend                      并行启动所有前端服务"
	@echo "    make stop-frontend                     停止所有前端服务（本地测试）"
	@echo ""
	@echo "  其他"
	@echo "    make list-apps                         列出所有可用的应用程序"
	@echo "    make help                              显示此帮助信息"
	@echo ""
	@echo "📝 Docker 镜像标签规则："
	@echo "   有 git tag  →  <tag>-<commit>   示例：demoapp:v1.0.0-abc1234"
	@echo "   无 git tag  →  <commit>         示例：demoapp:abc1234"
	@echo ""