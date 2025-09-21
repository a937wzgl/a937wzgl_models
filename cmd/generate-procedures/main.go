package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
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

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Name       string   `json:"name"`
	DSN        string   `json:"dsn"`
	OutPath    string   `json:"out_path"`
	Procedures []string `json:"procedures"`
}

// Config 完整配置
type Config struct {
	Databases []DatabaseConfig `json:"databases"`
}

func main() {
	// 读取配置文件
	configFile := "databases.yml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	fmt.Printf("从配置文件 %s 加载了 %d 个数据库配置\n", configFile, len(config.Databases))

	// 生成所有数据库的存储过程包装方法
	for _, dbConfig := range config.Databases {
		fmt.Printf("\n正在生成数据库 %s 的存储过程包装方法...\n", dbConfig.Name)
		err := generateProcedures(dbConfig)
		if err != nil {
			log.Printf("生成数据库 %s 的存储过程失败: %v", dbConfig.Name, err)
			continue
		}
		fmt.Printf("数据库 %s 的存储过程包装方法生成完成！\n", dbConfig.Name)
	}

	fmt.Println("\n所有数据库存储过程包装方法生成完成！")
}

// loadConfig 加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// generateProcedures 生成指定数据库的存储过程包装方法
func generateProcedures(dbConfig DatabaseConfig) error {
	// 连接数据库
	db, err := gorm.Open(mysql.Open(dbConfig.DSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 创建输出目录
	err = os.MkdirAll(dbConfig.OutPath, 0755)
	if err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 获取存储过程列表
	var procedures []ProcedureInfo
	if len(dbConfig.Procedures) > 0 {
		// 使用指定的存储过程
		fmt.Printf("使用指定的存储过程: %v\n", dbConfig.Procedures)
		for _, procName := range dbConfig.Procedures {
			proc, err := getProcedureInfo(db, dbConfig.Name, procName)
			if err != nil {
				fmt.Printf("警告: 无法获取存储过程 %s 的信息: %v\n", procName, err)
				continue
			}
			procedures = append(procedures, *proc)
		}
	} else {
		// 获取所有存储过程
		fmt.Printf("正在扫描数据库 %s 的存储过程...\n", dbConfig.Name)
		allProcedures, err := getAllProcedures(db, dbConfig.Name)
		if err != nil {
			return fmt.Errorf("获取存储过程列表失败: %v", err)
		}
		procedures = allProcedures
		fmt.Printf("数据库 %s 扫描完成，找到 %d 个存储过程\n", dbConfig.Name, len(procedures))
	}

	if len(procedures) == 0 {
		fmt.Printf("数据库 %s 中没有找到存储过程\n", dbConfig.Name)
		return nil
	}

	// 生成存储过程包装方法文件
	err = generateProcedureFile(dbConfig, procedures)
	if err != nil {
		return fmt.Errorf("生成存储过程文件失败: %v", err)
	}

	fmt.Printf("生成的文件位于: %s/procedures.gen.go\n", dbConfig.OutPath)
	return nil
}

// getAllProcedures 获取所有存储过程
func getAllProcedures(db *gorm.DB, dbName string) ([]ProcedureInfo, error) {
	var procedures []ProcedureInfo

	// 首先检查是否有存储过程
	var count int64
	countQuery := `
		SELECT COUNT(*) as count
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'PROCEDURE'
	`

	err := db.Raw(countQuery, dbName).Row().Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("查询存储过程数量失败: %v", err)
	}

	fmt.Printf("数据库 %s 中找到 %d 个存储过程\n", dbName, count)

	if count == 0 {
		return procedures, nil
	}

	// 查询存储过程信息
	query := `
		SELECT
			ROUTINE_NAME as name,
			COALESCE(ROUTINE_DEFINITION, '') as definition
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'PROCEDURE'
		ORDER BY ROUTINE_NAME
	`

	rows, err := db.Raw(query, dbName).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询存储过程信息失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var proc ProcedureInfo
		var definition sql.NullString

		err := rows.Scan(&proc.Name, &definition)
		if err != nil {
			fmt.Printf("警告: 扫描存储过程 %s 失败: %v\n", proc.Name, err)
			continue
		}

		if definition.Valid {
			proc.Definition = definition.String
		}

		// 解析参数信息
		proc.Parameters = parseProcedureParameters(proc.Definition)

		fmt.Printf("找到存储过程: %s (参数: %v)\n", proc.Name, proc.Parameters)
		procedures = append(procedures, proc)
	}

	return procedures, nil
}

// getProcedureInfo 获取指定存储过程的信息
func getProcedureInfo(db *gorm.DB, dbName, procName string) (*ProcedureInfo, error) {
	var proc ProcedureInfo
	query := `
		SELECT
			ROUTINE_NAME as name,
			ROUTINE_DEFINITION as definition
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ? AND ROUTINE_TYPE = 'PROCEDURE'
	`

	err := db.Raw(query, dbName, procName).Row().Scan(&proc.Name, &proc.Definition)
	if err != nil {
		return nil, err
	}

	// 解析参数信息
	proc.Parameters = parseProcedureParameters(proc.Definition)

	return &proc, nil
}

// parseProcedureParameters 解析存储过程的参数
func parseProcedureParameters(definition string) []string {
	var params []string

	if definition == "" {
		return params
	}

	// 首先尝试从参数表获取更准确的信息
	// 这里我们简化处理，直接返回空参数列表
	// 在实际使用中，可以通过查询 information_schema.PARAMETERS 表获取更准确的参数信息

	// 简单的参数解析，从CREATE PROCEDURE开始找到括号内的参数
	upperDef := strings.ToUpper(definition)
	start := strings.Index(upperDef, "PROCEDURE")
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
	paramStr = strings.TrimSpace(paramStr)
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
				// 找到参数名（通常是第二个词，跳过 IN/OUT/INOUT）
				for i, word := range words {
					if i > 0 && !strings.EqualFold(word, "IN") &&
						!strings.EqualFold(word, "OUT") &&
						!strings.EqualFold(word, "INOUT") {
						// 清理参数名（移除可能的类型信息）
						paramName := strings.Split(word, " ")[0]
						params = append(params, paramName)
						break
					}
				}
			}
		}
	}

	return params
}

// generateProcedureFile 生成存储过程包装方法文件
func generateProcedureFile(dbConfig DatabaseConfig, procedures []ProcedureInfo) error {
	var code strings.Builder

	// 生成包声明和导入
	code.WriteString("// Code generated by gorm.io/gen. DO NOT EDIT.\n")
	code.WriteString("// Code generated by gorm.io/gen. DO NOT EDIT.\n")
	code.WriteString("// Code generated by gorm.io/gen. DO NOT EDIT.\n\n")

	code.WriteString("package procedures\n\n")

	code.WriteString("import (\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"database/sql\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"gorm.io/gorm\"\n")
	code.WriteString(")\n\n")

	// 生成存储过程包装结构体
	code.WriteString("// ProcedureCaller 存储过程调用器\n")
	code.WriteString("type ProcedureCaller struct {\n")
	code.WriteString("\tdb *gorm.DB\n")
	code.WriteString("}\n\n")

	// 生成构造函数
	code.WriteString("// NewProcedureCaller 创建存储过程调用器\n")
	code.WriteString("func NewProcedureCaller(db *gorm.DB) *ProcedureCaller {\n")
	code.WriteString("\treturn &ProcedureCaller{db: db}\n")
	code.WriteString("}\n\n")

	// 生成每个存储过程的包装方法
	for _, proc := range procedures {
		generateProcedureMethod(&code, proc)
	}

	// 生成事务支持的方法
	code.WriteString("// Transaction 执行事务中的存储过程\n")
	code.WriteString("func (pc *ProcedureCaller) Transaction(fc func(tx *ProcedureCaller) error, opts ...*sql.TxOptions) error {\n")
	code.WriteString("\treturn pc.db.Transaction(func(tx *gorm.DB) error {\n")
	code.WriteString("\t\treturn fc(NewProcedureCaller(tx))\n")
	code.WriteString("\t}, opts...)\n")
	code.WriteString("}\n\n")

	// 生成上下文支持的方法
	code.WriteString("// WithContext 设置上下文\n")
	code.WriteString("func (pc *ProcedureCaller) WithContext(ctx context.Context) *ProcedureCaller {\n")
	code.WriteString("\treturn &ProcedureCaller{db: pc.db.WithContext(ctx)}\n")
	code.WriteString("}\n")

	// 写入文件
	filePath := fmt.Sprintf("%s/procedures.gen.go", dbConfig.OutPath)
	return ioutil.WriteFile(filePath, []byte(code.String()), 0644)
}

// generateProcedureMethod 生成单个存储过程的包装方法
func generateProcedureMethod(code *strings.Builder, proc ProcedureInfo) {
	methodName := toCamelCase(proc.Name)

	// 生成方法注释
	code.WriteString(fmt.Sprintf("// %s 调用存储过程 %s\n", methodName, proc.Name))

	// 生成方法签名
	params := make([]string, len(proc.Parameters))
	paramTypes := make([]string, len(proc.Parameters))
	callParams := make([]string, len(proc.Parameters))

	for i, param := range proc.Parameters {
		params[i] = fmt.Sprintf("%s interface{}", toCamelCase(param))
		paramTypes[i] = "interface{}"
		callParams[i] = toCamelCase(param)
	}

	paramStr := strings.Join(params, ", ")
	if paramStr != "" {
		paramStr = ", " + paramStr
	}

	code.WriteString(fmt.Sprintf("func (pc *ProcedureCaller) %s(ctx context.Context%s) error {\n", methodName, paramStr))

	// 生成方法体
	if len(proc.Parameters) > 0 {
		code.WriteString(fmt.Sprintf("\treturn pc.db.WithContext(ctx).Exec(\"CALL %s(%s)\", %s).Error\n",
			proc.Name,
			strings.Repeat("?,", len(proc.Parameters)-1)+"?",
			strings.Join(callParams, ", ")))
	} else {
		code.WriteString(fmt.Sprintf("\treturn pc.db.WithContext(ctx).Exec(\"CALL %s()\").Error\n", proc.Name))
	}

	code.WriteString("}\n\n")

	// 生成返回结果的版本
	resultMethodName := methodName + "WithResult"
	code.WriteString(fmt.Sprintf("// %s 调用存储过程 %s 并返回结果\n", resultMethodName, proc.Name))
	code.WriteString(fmt.Sprintf("func (pc *ProcedureCaller) %s(ctx context.Context%s) ([]map[string]interface{}, error) {\n", resultMethodName, paramStr))

	code.WriteString("\tvar results []map[string]interface{}\n")

	if len(proc.Parameters) > 0 {
		code.WriteString(fmt.Sprintf("\trows, err := pc.db.WithContext(ctx).Raw(\"CALL %s(%s)\", %s).Rows()\n",
			proc.Name,
			strings.Repeat("?,", len(proc.Parameters)-1)+"?",
			strings.Join(callParams, ", ")))
	} else {
		code.WriteString(fmt.Sprintf("\trows, err := pc.db.WithContext(ctx).Raw(\"CALL %s()\").Rows()\n", proc.Name))
	}

	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn nil, err\n")
	code.WriteString("\t}\n")
	code.WriteString("\tdefer rows.Close()\n\n")

	code.WriteString("\tcolumns, err := rows.Columns()\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn nil, err\n")
	code.WriteString("\t}\n\n")

	code.WriteString("\tfor rows.Next() {\n")
	code.WriteString("\t\tvalues := make([]interface{}, len(columns))\n")
	code.WriteString("\t\tscanArgs := make([]interface{}, len(values))\n")
	code.WriteString("\t\tfor i := range values {\n")
	code.WriteString("\t\t\tscanArgs[i] = &values[i]\n")
	code.WriteString("\t\t}\n\n")
	code.WriteString("\t\terr = rows.Scan(scanArgs...)\n")
	code.WriteString("\t\tif err != nil {\n")
	code.WriteString("\t\t\treturn nil, err\n")
	code.WriteString("\t\t}\n\n")
	code.WriteString("\t\trow := make(map[string]interface{})\n")
	code.WriteString("\t\tfor i, col := range columns {\n")
	code.WriteString("\t\t\trow[col] = values[i]\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, row)\n")
	code.WriteString("\t}\n\n")
	code.WriteString("\treturn results, nil\n")
	code.WriteString("}\n\n")
}

// toCamelCase 转换为驼峰命名
func toCamelCase(str string) string {
	parts := strings.Split(str, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return strings.Join(parts, "")
}
