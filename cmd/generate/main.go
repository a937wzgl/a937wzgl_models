package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	// 获取命令行参数
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("使用方法:")
		fmt.Println("  go run cmd/generate/main.go <database_name>")
		fmt.Println("  go run cmd/generate/main.go all")
		fmt.Println("")
		fmt.Println("环境变量:")
		fmt.Println("  DB_DSN_<NAME> - 数据库连接字符串")
		fmt.Println("  DB_TABLES_<NAME> - 指定表名，用逗号分隔")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  export DB_DSN_USER='root:password@tcp(localhost:3306)/user_db?charset=utf8mb4&parseTime=True&loc=Local'")
		fmt.Println("  export DB_DSN_ORDER='root:password@tcp(localhost:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local'")
		fmt.Println("  go run cmd/generate/main.go user")
		fmt.Println("  go run cmd/generate/main.go order")
		fmt.Println("  go run cmd/generate/main.go all")
		return
	}

	databaseName := strings.ToUpper(args[0])

	if databaseName == "ALL" {
		// 生成所有配置的数据库
		generateAllDatabases()
	} else {
		// 生成指定数据库
		generateDatabase(databaseName)
	}
}

// generateAllDatabases 生成所有数据库的模型
func generateAllDatabases() {
	// 从环境变量中查找所有数据库配置
	envVars := os.Environ()
	var databases []string

	for _, env := range envVars {
		if strings.HasPrefix(env, "DB_DSN_") {
			dbName := strings.TrimPrefix(env, "DB_DSN_")
			databases = append(databases, dbName)
		}
	}

	if len(databases) == 0 {
		fmt.Println("未找到任何数据库配置，请设置 DB_DSN_<NAME> 环境变量")
		return
	}

	fmt.Printf("找到 %d 个数据库配置: %v\n", len(databases), databases)

	for _, dbName := range databases {
		fmt.Printf("\n正在生成数据库 %s 的模型...\n", dbName)
		generateDatabase(dbName)
	}
}

// generateDatabase 生成指定数据库的模型
func generateDatabase(dbName string) {
	// 从环境变量获取数据库连接信息
	dsnEnv := fmt.Sprintf("DB_DSN_%s", dbName)
	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		log.Fatalf("未找到数据库 %s 的连接配置，请设置环境变量 %s", dbName, dsnEnv)
	}

	// 获取指定表名
	tablesEnv := fmt.Sprintf("DB_TABLES_%s", dbName)
	tablesStr := os.Getenv(tablesEnv)
	var specificTables []string
	if tablesStr != "" {
		specificTables = strings.Split(tablesStr, ",")
		for i, table := range specificTables {
			specificTables[i] = strings.TrimSpace(table)
		}
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库 %s 失败: %v", dbName, err)
	}

	// 创建输出目录
	outPath := fmt.Sprintf("./models/%s", strings.ToLower(dbName))
	err = os.MkdirAll(outPath, 0755)
	if err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	// 创建生成器
	g := gen.NewGenerator(gen.Config{
		OutPath: outPath,
		Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,

		// 字段配置
		FieldWithIndexTag: true, // 为字段添加索引标签
		FieldWithTypeTag:  true, // 为字段添加类型标签
		FieldSignable:     true, // 生成可签名字段
		FieldNullable:     true, // 生成指针当字段可空
	})

	// 设置数据库连接
	g.UseDB(db)

	// 获取表名
	var tables []string
	if len(specificTables) > 0 {
		// 使用指定的表名
		tables = specificTables
		fmt.Printf("使用指定的表: %v\n", tables)
	} else {
		// 获取所有表名
		allTables, err := db.Migrator().GetTables()
		if err != nil {
			log.Fatalf("获取表列表失败: %v", err)
		}
		tables = allTables
		fmt.Printf("找到 %d 个表: %v\n", len(tables), tables)
	}

	if len(tables) == 0 {
		fmt.Printf("数据库 %s 中没有找到表\n", dbName)
		return
	}

	// 生成所有表的模型
	var models []interface{}
	for _, table := range tables {
		models = append(models, g.GenerateModel(table))
	}

	// 应用模型
	g.ApplyBasic(models...)

	// 执行生成
	g.Execute()

	fmt.Printf("数据库 %s 的模型生成完成！\n", dbName)
	fmt.Printf("生成的文件位于: %s/\n", outPath)
}
