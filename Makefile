# 定义变量
BINARY_NAME=podcast
BINARY_DIR=bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# 获取当前时间用于构建标签
BUILD_TIME=`date +%FT%T%z`

GOCMD			:=$(shell which go)

# Go modules
GO_BUILD=CGO_ENABLED=0 $(GOCMD) build -a -installsuffix cgo
IMPORT_PATH		:=podcast/cmd
BUILD_TIME		:=$(shell date "+%F %T")
COMMIT_ID       :=$(shell git rev-parse HEAD)
GO_VERSION      :=$(shell $(GOCMD) version)
VERSION			:=$(shell git describe --tags)
BUILD_USER		:=$(shell whoami)
FLAG			:="-X '${IMPORT_PATH}.buildTime=${BUILD_TIME}' -X '${IMPORT_PATH}.commitID=${COMMIT_ID}' -X '${IMPORT_PATH}.goVersion=${GO_VERSION}' -X '${IMPORT_PATH}.Version=${VERSION}' -X '${IMPORT_PATH}.buildUser=${BUILD_USER}'"

# 构建目标
.PHONY: build clean run

# 默认目标
all: build

# 创建 bin 目录并编译程序
build:
	@echo "开始编译..."
	@mkdir -p $(BINARY_DIR)
	$(GO_BUILD) -o $(BINARY_PATH) -ldflags $(FLAG)  main.go
	@upx --lzma $(BINARY_PATH)
	@echo "编译完成，二进制文件位于 $(BINARY_PATH)"

# 清理生成的二进制文件
clean:
	@echo "清理编译产物..."
	@rm -rf $(BINARY_DIR)
	@echo "清理完成"

# 运行程序
run:
	@make build
	@echo "运行程序..."
	./$(BINARY_PATH)


# 前端编译
frontend-test:
	@cd frontend  && npm run build:test
	@cp -r frontend/dist/* ./web/dist
frontend-pro:
	@cd frontend  && npm run build:pro&&rm -rf dist/log &&cd ..
	@rm -rf ./web/dist/assets  ./web/dist/index.html && cp -r frontend/dist/* ./web/dist
check:
	@golangci-lint run ./...