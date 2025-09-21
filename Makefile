# GORM 模型生成 Makefile

.PHONY: help install generate generate-multi generate-single generate-procedures clean scan

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  install             - 安装依赖"
	@echo "  scan                - 扫描 MySQL 服务器上的所有数据库"
	@echo "  generate            - 生成所有数据库模型 (使用配置文件)"
	@echo "  generate-multi      - 生成所有数据库模型 (使用多数据库配置)"
	@echo "  generate-single     - 生成单个数据库模型"
	@echo "  generate-procedures - 生成存储过程包装方法"
	@echo "  clean               - 清理生成的文件"
	@echo "  help                - 显示此帮助信息"
	@echo ""
	@echo "多数据库环境变量设置:"
	@echo "  export DB_DSN_USER='root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local'"
	@echo "  export DB_DSN_ORDER='root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local'"
	@echo "  export DB_TABLES_USER='users,profiles'  # 可选，指定特定表"
	@echo ""
	@echo "示例:"
	@echo "  make generate-single DB=user"
	@echo "  make generate-single DB=order"
	@echo "  make generate-multi"

# 安装依赖
install:
	@echo "安装 GORM Gen 工具..."
	go install gorm.io/gen/tools/gentool@latest
	@echo "安装 YAML 解析库..."
	go get gopkg.in/yaml.v2
	@echo "安装项目依赖..."
	go mod tidy

# 生成模型 - 使用单数据库配置文件
generate:
	@echo "使用单数据库配置文件生成模型..."
	gentool -c gen.yml

# 生成模型 - 使用多数据库配置文件
generate-multi:
	@echo "使用多数据库配置文件生成模型..."
	go run cmd/generate-multi/main.go databases.yml

# 生成存储过程包装方法
generate-procedures:
	@echo "生成存储过程包装方法..."
	go run cmd/generate-procedures/main.go databases.yml

# 生成模型 - 使用多数据库配置文件 (指定配置文件)
generate-multi-config:
	@echo "使用指定配置文件生成模型..."
	@if [ -z "$(CONFIG)" ]; then echo "错误: 请指定 CONFIG 参数，例如: make generate-multi-config CONFIG=my-databases.yml"; exit 1; fi
	go run cmd/generate-multi/main.go $(CONFIG)

# 生成模型 - 单个数据库 (环境变量方式)
generate-single:
	@echo "生成单个数据库模型..."
	@if [ -z "$(DB)" ]; then echo "错误: 请指定 DB 参数，例如: make generate-single DB=user"; exit 1; fi
	go run cmd/generate/main.go $(DB)

# 生成模型 - 所有数据库 (环境变量方式)
generate-all:
	@echo "生成所有数据库模型..."
	go run cmd/generate/main.go all

# 生成模型 - 指定表
generate-tables:
	@echo "请指定表名，例如: make generate-tables TABLES=users,posts"
	@if [ -z "$(TABLES)" ]; then echo "错误: 请指定 TABLES 参数"; exit 1; fi
	gentool -dsn "$(DB_DSN)" -tables "$(TABLES)" -outPath "./models"

# 清理生成的文件
clean:
	@echo "清理生成的文件..."
	rm -rf models/*/
	rm -rf models/*.go
	@echo "清理完成"

# 设置环境变量示例
env-example:
	@echo "单数据库环境变量设置:"
	@echo "export DB_DSN=\"root:password@tcp(localhost:3306)/your_database?charset=utf8mb4&parseTime=True&loc=Local\""
	@echo ""
	@echo "多数据库环境变量设置:"
	@echo "export DB_DSN_USER='root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local'"
	@echo "export DB_DSN_ORDER='root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local'"
	@echo "export DB_DSN_PRODUCT='root:password@tcp(localhost:3306)/product_db?charset=utf8mb4&parseTime=True&loc=Local'"
	@echo "export DB_TABLES_USER='users,profiles'  # 可选，指定特定表"
	@echo "export DB_TABLES_ORDER='orders,order_items'  # 可选，指定特定表"

# 扫描数据库
scan:
	@echo "扫描 MySQL 服务器上的所有数据库..."
	@if [ -z "$(HOST)" ]; then echo "错误: 请指定 HOST 参数，例如: make scan HOST=127.0.0.1 PORT=3306 USER=root PASSWORD=root123"; exit 1; fi
	go run cmd/scan/main.go $(HOST) $(PORT) $(USER) $(PASSWORD)

# 扫描数据库 - 使用环境变量
scan-env:
	@echo "使用环境变量扫描数据库..."
	@if [ -z "$$DB_HOST" ]; then echo "错误: 请设置环境变量 DB_HOST, DB_PORT, DB_USER, DB_PASSWORD"; exit 1; fi
	go run cmd/scan/main.go

# 查看帮助
help-generate:
	@echo "生成器帮助信息:"
	go run cmd/generate/main.go

help-scan:
	@echo "扫描工具帮助信息:"
	go run cmd/scan/main.go

help-procedures:
	@echo "存储过程生成器帮助信息:"
	go run cmd/generate-procedures/main.go

