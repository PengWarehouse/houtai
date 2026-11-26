APP := goProject
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP)
PKG := ./...

.PHONY: help run build test vet fmt tidy migrate clean

help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

run: ## 启动服务
	go run ./cmd/server

build: ## 编译到 bin/
	go build -trimpath -o $(BIN) ./cmd/server

test: ## 运行全部测试
	go test $(PKG)

vet: ## 静态检查
	go vet $(PKG)

fmt: ## 格式化代码
	gofmt -w .

tidy: ## 整理依赖
	go mod tidy

migrate: ## 执行 db/schema.sql + db/seed.sql
	go run ./cmd/migrate

clean: ## 清理编译产物
	rm -rf $(BIN_DIR)
