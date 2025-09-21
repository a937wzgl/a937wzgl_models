package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProcedureInfo 存储过程信息
type ProcedureInfo struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	ReturnType string   `json:"return_type"`
	Definition string   `json:"definition"`
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
	Name       string          `json:"name"`
	Tables     []string        `json:"tables"`
	Procedures []ProcedureInfo `json:"procedures"`
}

func main() {
	// 获取命令行参数
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("使用方法:")
		fmt.Println("  go run cmd/scan/main.go <host> <port> <user> <password>")
		fmt.Println("  go run cmd/scan/main.go 127.0.0.1 3306 root root123")
		fmt.Println("")
		fmt.Println("环境变量方式:")
		fmt.Println("  export DB_HOST=127.0.0.1")
		fmt.Println("  export DB_PORT=3306")
		fmt.Println("  export DB_USER=root")
		fmt.Println("  export DB_PASSWORD=root123")
		fmt.Println("  go run cmd/scan/main.go")
		return
	}

	var host, port, user, password string

	if len(args) >= 4 {
		// 使用命令行参数
		host = args[0]
		port = args[1]
		user = args[2]
		password = args[3]
	} else {
		// 使用环境变量
		host = getEnvOrDefault("DB_HOST", "127.0.0.1")
		port = getEnvOrDefault("DB_PORT", "3306")
		user = getEnvOrDefault("DB_USER", "root")
		password = getEnvOrDefault("DB_PASSWORD", "")
	}

	// 扫描数据库
	databases, err := scanDatabases(host, port, user, password)
	if err != nil {
		log.Fatalf("扫描数据库失败: %v", err)
	}

	// 输出结果
	fmt.Printf("连接到 MySQL 服务器: %s:%s\n", host, port)
	fmt.Printf("找到 %d 个数据库:\n\n", len(databases))

	for i, db := range databases {
		fmt.Printf("%d. 数据库: %s\n", i+1, db.Name)

		if len(db.Tables) > 0 {
			fmt.Printf("   表数量: %d\n", len(db.Tables))
			if len(db.Tables) <= 10 {
				fmt.Printf("   表名: %s\n", strings.Join(db.Tables, ", "))
			} else {
				fmt.Printf("   表名: %s ... (还有 %d 个表)\n",
					strings.Join(db.Tables[:10], ", "), len(db.Tables)-10)
			}
		} else {
			fmt.Printf("   表数量: 0 (空数据库)\n")
		}

		if len(db.Procedures) > 0 {
			fmt.Printf("   存储过程数量: %d\n", len(db.Procedures))
			for j, proc := range db.Procedures {
				if j >= 5 { // 只显示前5个存储过程
					fmt.Printf("   ... (还有 %d 个存储过程)\n", len(db.Procedures)-5)
					break
				}
				fmt.Printf("   存储过程: %s", proc.Name)
				if len(proc.Parameters) > 0 {
					fmt.Printf(" (参数: %s)", strings.Join(proc.Parameters, ", "))
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("   存储过程数量: 0\n")
		}
		fmt.Println()
	}

	// 生成环境变量命令
	fmt.Println("生成环境变量命令:")
	fmt.Println("```bash")
	for _, db := range databases {
		if len(db.Tables) > 0 {
			dbName := strings.ToUpper(db.Name)
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user, password, host, port, db.Name)
			fmt.Printf("export DB_DSN_%s=\"%s\"\n", dbName, dsn)
		}
	}
	fmt.Println("```")

	// 生成配置文件
	fmt.Println("\n生成 databases.yml 配置:")
	fmt.Println("```yaml")
	fmt.Println("databases:")
	for _, db := range databases {
		if len(db.Tables) > 0 {
			fmt.Printf("  - name: \"%s\"\n", strings.ToUpper(db.Name))
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user, password, host, port, db.Name)
			fmt.Printf("    dsn: \"%s\"\n", dsn)
			fmt.Printf("    out_path: \"./models/%s\"\n", db.Name)
			fmt.Printf("    tables: []  # 空数组表示生成所有表\n")
			fmt.Println()
		}
	}
	fmt.Println("global:")
	fmt.Println("  mode: \"without_context|with_default_query|with_query_interface\"")
	fmt.Println("  field_with_index_tag: true")
	fmt.Println("  field_with_type_tag: true")
	fmt.Println("  field_signable: true")
	fmt.Println("  field_with_null_tag: true")
	fmt.Println("```")
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// scanDatabases 扫描数据库
func scanDatabases(host, port, user, password string) ([]DatabaseInfo, error) {
	// 连接到 MySQL 服务器（不指定数据库）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 获取所有数据库名
	var databases []string
	err = db.Raw("SHOW DATABASES").Scan(&databases).Error
	if err != nil {
		return nil, fmt.Errorf("获取数据库列表失败: %v", err)
	}

	// 过滤掉系统数据库
	var filteredDatabases []string
	for _, dbName := range databases {
		if !isSystemDatabase(dbName) {
			filteredDatabases = append(filteredDatabases, dbName)
		}
	}

	// 获取每个数据库的表和存储过程信息
	var result []DatabaseInfo
	for _, dbName := range filteredDatabases {
		tables, err := getTablesInDatabase(host, port, user, password, dbName)
		if err != nil {
			fmt.Printf("警告: 无法获取数据库 %s 的表信息: %v\n", dbName, err)
			continue
		}

		procedures, err := getProceduresInDatabase(host, port, user, password, dbName)
		if err != nil {
			fmt.Printf("警告: 无法获取数据库 %s 的存储过程信息: %v\n", dbName, err)
		}

		result = append(result, DatabaseInfo{
			Name:       dbName,
			Tables:     tables,
			Procedures: procedures,
		})
	}

	return result, nil
}

// getTablesInDatabase 获取指定数据库中的表
func getTablesInDatabase(host, port, user, password, dbName string) ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	var tables []string
	err = db.Raw("SHOW TABLES").Scan(&tables).Error
	if err != nil {
		return nil, err
	}

	return tables, nil
}

// getProceduresInDatabase 获取指定数据库中的存储过程
func getProceduresInDatabase(host, port, user, password, dbName string) ([]ProcedureInfo, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 查询存储过程信息
	var procedures []ProcedureInfo
	query := `
		SELECT
			ROUTINE_NAME as name,
			ROUTINE_DEFINITION as definition
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'PROCEDURE'
		ORDER BY ROUTINE_NAME
	`

	rows, err := db.Raw(query, dbName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var proc ProcedureInfo
		err := rows.Scan(&proc.Name, &proc.Definition)
		if err != nil {
			continue
		}

		// 解析参数信息
		proc.Parameters = parseProcedureParameters(proc.Definition)

		procedures = append(procedures, proc)
	}

	return procedures, nil
}

// parseProcedureParameters 解析存储过程的参数
func parseProcedureParameters(definition string) []string {
	var params []string

	// 简单的参数解析，从CREATE PROCEDURE开始找到括号内的参数
	start := strings.Index(strings.ToUpper(definition), "PROCEDURE")
	if start == -1 {
		return params
	}

	// 找到第一个左括号
	leftParen := strings.Index(definition[start:], "(")
	if leftParen == -1 {
		return params
	}

	// 找到对应的右括号
	rightParen := strings.Index(definition[start+leftParen:], ")")
	if rightParen == -1 {
		return params
	}

	paramStr := definition[start+leftParen+1 : start+leftParen+rightParen]
	if paramStr == "" {
		return params
	}

	// 按逗号分割参数
	paramParts := strings.Split(paramStr, ",")
	for _, part := range paramParts {
		part = strings.TrimSpace(part)
		if part != "" {
			// 提取参数名（IN/OUT/INOUT param_name TYPE）
			words := strings.Fields(part)
			if len(words) >= 2 {
				// 找到参数名（通常是第二个词）
				for i, word := range words {
					if i > 0 && !strings.EqualFold(word, "IN") &&
						!strings.EqualFold(word, "OUT") &&
						!strings.EqualFold(word, "INOUT") {
						params = append(params, word)
						break
					}
				}
			}
		}
	}

	return params
}

// isSystemDatabase 判断是否为系统数据库
func isSystemDatabase(dbName string) bool {
	systemDBs := []string{
		"information_schema",
		"performance_schema",
		"mysql",
		"sys",
		"test",
	}

	for _, sysDB := range systemDBs {
		if strings.ToLower(dbName) == sysDB {
			return true
		}
	}

	return false
}
