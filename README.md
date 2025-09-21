# a937wzgl_models

基于 GORM 的模型生成工具，支持从 MySQL 数据库自动生成 Go 模型结构体。

## 功能特性

- 🚀 自动从数据库表生成 Go 模型
- 🔧 支持多种数据库驱动 (MySQL, PostgreSQL, SQLite)
- 📝 自动生成 CRUD 方法
- 🏷️ 智能字段标签生成
- ⚙️ 灵活的配置选项
- 🛠️ 多种生成方式
- 🗄️ **支持多数据库分离生成**
- 📁 **按数据库分目录组织模型文件**

## 快速开始

### 1. 安装依赖

```bash
make install
```

### 2. 扫描数据库（可选）

在生成模型之前，您可以先扫描 MySQL 服务器上的所有数据库：

```bash
# 使用命令行参数
make scan HOST=127.0.0.1 PORT=3306 USER=root PASSWORD=root123

# 使用环境变量
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root123
make scan-env
```

扫描工具会：
- 列出所有非系统数据库
- 显示每个数据库的表数量
- 自动生成环境变量命令
- 自动生成 `databases.yml` 配置文件

### 3. 配置数据库连接

#### 方式一：环境变量
```bash
export DB_DSN="root:password@tcp(localhost:3306)/your_database?charset=utf8mb4&parseTime=True&loc=Local"
```

#### 方式二：修改配置文件
编辑 `gen.yml` 文件中的数据库连接信息：
```yaml
database:
  driver: mysql
  source: "root:password@tcp(localhost:3306)/your_database?charset=utf8mb4&parseTime=True&loc=Local"
```

### 4. 生成模型

#### 方式一：多数据库配置文件生成
```bash
# 使用多数据库配置文件生成所有数据库模型
make generate-multi

# 使用自定义配置文件
make generate-multi-config CONFIG=my-databases.yml
```

#### 方式二：环境变量方式生成
```bash
# 设置环境变量
export DB_DSN_USER='root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local'
export DB_DSN_ORDER='root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local'

# 生成单个数据库
make generate-single DB=user
make generate-single DB=order

# 生成所有数据库
make generate-all
```

#### 方式三：单数据库生成
```bash
# 使用单数据库配置文件
make generate

# 生成指定表
make generate-tables TABLES=users,posts,comments
```

## 使用方法

### 命令行工具

#### 安装 gentool
```bash
go install gorm.io/gen/tools/gentool@latest
```

#### 基本用法
```bash
# 生成所有表
gentool -dsn "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local" -outPath "./models"

# 生成指定表
gentool -dsn "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local" -tables "users,posts" -outPath "./models"

# 使用配置文件
gentool -c gen.yml
```

### 编程方式

```go
package main

import (
    "gorm.io/driver/mysql"
    "gorm.io/gen"
    "gorm.io/gorm"
)

func main() {
    db, _ := gorm.Open(mysql.Open("dsn"))
    g := gen.NewGenerator(gen.Config{
        OutPath: "./models",
        Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
    })
    g.UseDB(db)
    g.ApplyBasic(g.GenerateAllTable()...)
    g.Execute()
}
```

## 配置选项

### 多数据库配置 (databases.yml)

```yaml
# 多数据库配置文件
databases:
  # 用户数据库
  - name: "USER"
    dsn: "root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/user"
    tables: []  # 空数组表示生成所有表
    # tables: ["users", "profiles", "sessions"]  # 指定特定表

  # 订单数据库
  - name: "ORDER"
    dsn: "root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/order"
    tables: []  # 空数组表示生成所有表

  # 商品数据库
  - name: "PRODUCT"
    dsn: "root:password@tcp(localhost:3306)/product_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/product"
    tables: []  # 空数组表示生成所有表

# 全局配置
global:
  mode: "without_context|with_default_query|with_query_interface"
  field_with_index_tag: true
  field_with_type_tag: true
  field_signable: true
  field_with_null_tag: true
```

### 单数据库配置 (gen.yml)

```yaml
# 数据库配置
database:
  driver: mysql
  source: "连接字符串"

# 输出配置
outPath: "./models"        # 输出目录
outFile: "gen.go"         # 输出文件名
package: "models"         # 包名

# 生成模式
mode: "without_context|with_default_query|with_query_interface"

# 字段配置
fieldWithIndexTag: true   # 为字段添加索引标签
fieldWithTypeTag: true    # 为字段添加类型标签
fieldSignable: true       # 生成可签名字段
fieldWithNullTag: true    # 为字段添加 null 标签

# 表配置
tables:                   # 指定表名，留空则生成所有表
  - users
  - posts
```

### 生成模式说明

- `without_context`: 不使用 context
- `with_default_query`: 生成默认查询方法
- `with_query_interface`: 生成查询接口

## 项目结构

```
a937wzgl_models/
├── cmd/
│   ├── generate/
│   │   └── main.go          # 单数据库生成器
│   └── generate-multi/
│       └── main.go          # 多数据库生成器
├── models/                  # 生成的模型文件
│   ├── user/               # 用户数据库模型
│   ├── order/              # 订单数据库模型
│   ├── product/            # 商品数据库模型
│   └── log/                # 日志数据库模型
├── databases.yml           # 多数据库配置文件
├── gen.yml                 # 单数据库配置文件
├── Makefile                # 构建脚本
├── go.mod                  # Go 模块文件
└── README.md               # 说明文档
```

## 常用命令

```bash
# 查看帮助
make help

# 安装依赖
make install

# 扫描数据库
make scan HOST=127.0.0.1 PORT=3306 USER=root PASSWORD=root123
make scan-env  # 使用环境变量

# 生成所有表模型
make generate

# 生成指定表模型
make generate-tables TABLES=users,posts

# 清理生成的文件
make clean

# 设置环境变量示例
make env-example
```

## 数据库扫描功能

### 扫描工具特性

- 🔍 **自动发现数据库**：扫描 MySQL 服务器上的所有非系统数据库
- 📊 **表信息统计**：显示每个数据库的表数量和表名
- ⚙️ **自动生成配置**：自动生成环境变量和配置文件
- 🚫 **过滤系统库**：自动过滤 `information_schema`、`mysql`、`sys` 等系统数据库

### 使用方法

#### 命令行参数方式
```bash
make scan HOST=127.0.0.1 PORT=3306 USER=root PASSWORD=root123
```

#### 环境变量方式
```bash
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root123
make scan-env
```

#### 直接运行
```bash
go run cmd/scan/main.go 127.0.0.1 3306 root root123
```

### 扫描结果示例

```
连接到 MySQL 服务器: 127.0.0.1:3306
找到 3 个数据库:

1. 数据库: user_management
   表数量: 5
   表名: users, profiles, sessions, roles, permissions

2. 数据库: order_system
   表数量: 8
   表名: orders, order_items, payments, shipping, customers, products, categories, inventory

3. 数据库: analytics
   表数量: 3
   表名: page_views, user_events, conversion_tracking

生成环境变量命令:
```bash
export DB_DSN_USER_MANAGEMENT="root:root123@tcp(127.0.0.1:3306)/user_management?charset=utf8mb4&parseTime=True&loc=Local"
export DB_DSN_ORDER_SYSTEM="root:root123@tcp(127.0.0.1:3306)/order_system?charset=utf8mb4&parseTime=True&loc=Local"
export DB_DSN_ANALYTICS="root:root123@tcp(127.0.0.1:3306)/analytics?charset=utf8mb4&parseTime=True&loc=Local"
```

生成 databases.yml 配置:
```yaml
databases:
  - name: "USER_MANAGEMENT"
    dsn: "root:root123@tcp(127.0.0.1:3306)/user_management?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/user_management"
    tables: []  # 空数组表示生成所有表

  - name: "ORDER_SYSTEM"
    dsn: "root:root123@tcp(127.0.0.1:3306)/order_system?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/order_system"
    tables: []  # 空数组表示生成所有表

  - name: "ANALYTICS"
    dsn: "root:root123@tcp(127.0.0.1:3306)/analytics?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/analytics"
    tables: []  # 空数组表示生成所有表

global:
  mode: "without_context|with_default_query|with_query_interface"
  field_with_index_tag: true
  field_with_type_tag: true
  field_signable: true
  field_with_null_tag: true
```
```

## 高级用法

### 自定义字段映射

```go
g.ApplyBasic(
    g.GenerateModel("users", 
        gen.FieldType("id", "int64"),
        gen.FieldGORMTag("username", `gorm:"size:50;uniqueIndex"`),
    ),
)
```

### 生成 CRUD 方法

```go
g.ApplyInterface(func(method gen.Method) {
    // 自定义方法
}, g.GenerateModel("users"))
```

### 字段标签配置

- `fieldWithIndexTag`: 为字段添加索引标签
- `fieldWithTypeTag`: 为字段添加类型标签  
- `fieldSignable`: 生成可签名字段
- `fieldWithNullTag`: 为字段添加 null 标签

## 注意事项

1. 确保数据库连接信息正确
2. 生成前建议备份现有模型文件
3. 可以根据需要调整配置文件中的选项
4. 生成的模型文件会覆盖同名的现有文件

## 依赖

- Go 1.25.1+
- gorm.io/gorm
- gorm.io/driver/mysql
- gorm.io/gen

## 多数据库支持

### 环境变量方式

#### 设置多个数据库连接
```bash
# 用户数据库
export DB_DSN_USER="root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local"

# 订单数据库
export DB_DSN_ORDER="root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local"

# 商品数据库
export DB_DSN_PRODUCT="root:password@tcp(localhost:3306)/product_db?charset=utf8mb4&parseTime=True&loc=Local"

# 日志数据库
export DB_DSN_LOG="root:password@tcp(localhost:3306)/log_db?charset=utf8mb4&parseTime=True&loc=Local"
```

#### 指定特定表（可选）
```bash
# 只生成指定表的模型
export DB_TABLES_USER="users,profiles,sessions"
export DB_TABLES_ORDER="orders,order_items,payments"
export DB_TABLES_PRODUCT="products,categories,inventory"
```

#### 生成命令
```bash
# 生成单个数据库
make generate-single DB=user
make generate-single DB=order

# 生成所有数据库
make generate-all
```

### 配置文件方式

#### 编辑 databases.yml
```yaml
databases:
  - name: "USER"
    dsn: "root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/user"
    tables: []  # 空数组表示生成所有表

  - name: "ORDER"
    dsn: "root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/order"
    tables: ["orders", "order_items", "payments"]  # 指定特定表

  - name: "PRODUCT"
    dsn: "root:password@tcp(localhost:3306)/product_db?charset=utf8mb4&parseTime=True&loc=Local"
    out_path: "./models/product"
    tables: []  # 空数组表示生成所有表

global:
  mode: "without_context|with_default_query|with_query_interface"
  field_with_index_tag: true
  field_with_type_tag: true
  field_signable: true
  field_with_null_tag: true
```

#### 生成命令
```bash
# 使用默认配置文件
make generate-multi

# 使用自定义配置文件
make generate-multi-config CONFIG=my-databases.yml
```

### 输出目录结构

使用多数据库配置后，模型文件会按数据库分目录组织：

```
models/
├── user/
│   ├── gen.go
│   ├── user.gen.go
│   └── profile.gen.go
├── order/
│   ├── gen.go
│   ├── order.gen.go
│   └── order_item.gen.go
└── product/
    ├── gen.go
    ├── product.gen.go
    └── category.gen.go
```

### 使用生成的模型

```go
package main

import (
    "github.com/a937wzgl/a937wzgl_models/models/user"
    "github.com/a937wzgl/a937wzgl_models/models/order"
    "github.com/a937wzgl/a937wzgl_models/models/product"
)

func main() {
    // 使用用户模型
    userQuery := user.Use(db)
    users, err := userQuery.Find()
    
    // 使用订单模型
    orderQuery := order.Use(db)
    orders, err := orderQuery.Find()
    
    // 使用商品模型
    productQuery := product.Use(db)
    products, err := productQuery.Find()
}
```

## 许可证

MIT License