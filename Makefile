.PHONY: all build frontend backend clean dev dev-frontend dev-backend install build-linux deploy

BINARY_NAME := nginx-certs-manager
BUILD_DIR := ./build
WEB_DIR := ./web
WEB_DIST := $(WEB_DIR)/dist
REMOTE_HOST := user@your-server-ip
REMOTE_DIR := /opt/nginx-certs-manager

all: build

# ========== 本地构建 ==========

# 构建当前平台
build: frontend backend
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

frontend:
	@echo "Building frontend..."
	cd $(WEB_DIR) && npm install && npm run build

backend:
	@echo "Building backend..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/

# ========== 交叉编译 (Windows -> Linux) ==========

# 在 Windows 上编译出 Linux 二进制 + 打包前端，一键部署包
build-linux: frontend
	@echo "Cross-compiling for Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/
	@echo ""
	@echo "=== 构建完成 ==="
	@echo "二进制: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "前端:   $(WEB_DIST)/"
	@echo ""
	@echo "部署方式:"
	@echo "  1. scp $(BUILD_DIR)/$(BINARY_NAME) $(REMOTE_HOST):$(REMOTE_DIR)/"
	@echo "  2. scp -r $(WEB_DIST) $(REMOTE_HOST):$(REMOTE_DIR)/web-dist"
	@echo "  3. ssh $(REMOTE_HOST) 'systemctl restart nginx-certs-manager'"

# 打包部署包 (tar.gz)
package: frontend
	@echo "Packaging for Linux deployment..."
	GOOS=linux GOARCH=amd64 go build -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/
	tar -czf $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME) -C ../$(WEB_DIST) .
	@echo "Package: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz"

# ========== 开发 ==========

dev: dev-backend

dev-backend:
	@echo "Starting backend (port 8080)..."
	WEB_DIR=$(WEB_DIST) go run ./cmd/server/

dev-frontend:
	@echo "Starting frontend dev server (port 3000)..."
	cd $(WEB_DIR) && npm run dev

# ========== 清理 ==========

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_DIST)
	rm -rf $(WEB_DIR)/node_modules
