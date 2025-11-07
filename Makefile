.PHONY: help build run clean test migrate docker-build docker-up docker-down docker-logs pre-commit-install pre-commit-run fmt openapi-validate openapi-ui

help:
	@echo "可用命令:"
	@echo "  make build              - 编译应用"
	@echo "  make run                - 运行应用"
	@echo "  make migrate            - 执行数据库迁移"
	@echo "  make clean              - 清理构建产物"
	@echo "  make test               - 运行测试"
	@echo "  make fmt                - 格式化 Go 代码（gofmt + goimports）"
	@echo "  make docker-build       - 构建 Docker 镜像"
	@echo "  make docker-up          - 启动所有服务"
	@echo "  make docker-down        - 停止所有服务"
	@echo "  make docker-logs        - 查看服务日志"
	@echo "  make pre-commit-install - 安装 pre-commit hooks"
	@echo "  make pre-commit-run     - 手动运行 pre-commit 检查"
	@echo "  make openapi-validate   - 验证 OpenAPI 文档格式"
	@echo "  make openapi-ui         - 启动 Swagger UI 预览 API 文档"

build:
	@echo "Building application..."
	@go build -o main cmd/server/main.go
	@echo "Build complete: ./main"

run:
	@echo "Starting application..."
	@go run cmd/server/main.go

migrate:
	@echo "执行数据库迁移..."
	@go run cmd/migrate/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -f main
	@rm -rf tmp/
	@echo "Clean complete"

test:
	@echo "Running tests..."
	@go test -v ./...

fmt:
	@echo "Formatting Go code..."
	@gofmt -w -s .
	@if [ -f $(HOME)/go/bin/goimports ]; then \
		echo "Running goimports..."; \
		$(HOME)/go/bin/goimports -w .; \
	elif command -v goimports &> /dev/null; then \
		echo "Running goimports..."; \
		goimports -w .; \
	else \
		echo "goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi
	@echo "Code formatting complete!"

docker-build:
	@echo "Building Docker image..."
	@docker build -t microvibe-go:latest .

docker-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d
	@echo "Services started. Use 'make docker-logs' to view logs"

docker-down:
	@echo "Stopping services..."
	@docker-compose down
	@echo "Services stopped"

docker-logs:
	@docker-compose logs -f

pre-commit-install:
	@echo "Installing pre-commit hooks..."
	@if ! command -v pre-commit &> /dev/null; then \
		echo "Error: pre-commit is not installed."; \
		echo "Please install it first:"; \
		echo "  - macOS: brew install pre-commit"; \
		echo "  - Linux: pip install pre-commit"; \
		exit 1; \
	fi
	@pre-commit install --install-hooks
	@pre-commit install --hook-type commit-msg
	@echo "Pre-commit hooks installed successfully!"

pre-commit-run:
	@echo "Running pre-commit checks on all files..."
	@pre-commit run --all-files

openapi-validate:
	@echo "Validating OpenAPI document..."
	@if ! command -v npx &> /dev/null; then \
		echo "Error: npx (Node.js) is not installed."; \
		echo "Please install Node.js first: https://nodejs.org/"; \
		exit 1; \
	fi
	@echo "Running OpenAPI validation (warnings are informational only)..."
	@npx @redocly/cli lint openapi.json --skip-rule security-defined --skip-rule operation-operationId --skip-rule operation-4xx-response || true
	@echo "✅ OpenAPI validation complete! (Check output above for any critical errors)"

openapi-ui:
	@echo "Starting Swagger UI on http://localhost:8081"
	@echo "Press Ctrl+C to stop..."
	@if ! command -v docker &> /dev/null; then \
		echo "Error: Docker is not installed."; \
		echo "Please install Docker first: https://www.docker.com/"; \
		exit 1; \
	fi
	@docker run --rm -p 8081:8080 \
		-e SWAGGER_JSON=/app/openapi.json \
		-v $(PWD)/openapi.json:/app/openapi.json \
		swaggerapi/swagger-ui
