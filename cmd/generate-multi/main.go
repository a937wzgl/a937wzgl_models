package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Name    string   `yaml:"name"`
	DSN     string   `yaml:"dsn"`
	OutPath string   `yaml:"out_path"`
	Tables  []string `yaml:"tables"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Mode              string `yaml:"mode"`
	FieldWithIndexTag bool   `yaml:"field_with_index_tag"`
	FieldWithTypeTag  bool   `yaml:"field_with_type_tag"`
	FieldSignable     bool   `yaml:"field_signable"`
	FieldNullable     bool   `yaml:"field_nullable"`
}

// Config 完整配置
type Config struct {
	Databases []DatabaseConfig `yaml:"databases"`
	Global    GlobalConfig     `yaml:"global"`
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

	// 生成所有数据库的模型
	for _, dbConfig := range config.Databases {
		fmt.Printf("\n正在生成数据库 %s 的模型...\n", dbConfig.Name)
		err := generateDatabase(dbConfig, config.Global)
		if err != nil {
			log.Printf("生成数据库 %s 失败: %v", dbConfig.Name, err)
			continue
		}
		fmt.Printf("数据库 %s 的模型生成完成！\n", dbConfig.Name)
	}

	fmt.Println("\n所有数据库模型生成完成！")
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

// generateDatabase 生成指定数据库的模型
func generateDatabase(dbConfig DatabaseConfig, globalConfig GlobalConfig) error {
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

	// 创建生成器
	g := gen.NewGenerator(gen.Config{
		OutPath: dbConfig.OutPath,
		Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,

		// 字段配置
		FieldWithIndexTag: globalConfig.FieldWithIndexTag,
		FieldWithTypeTag:  globalConfig.FieldWithTypeTag,
		FieldSignable:     globalConfig.FieldSignable,
		FieldNullable:     globalConfig.FieldNullable,
	})

	// 设置数据库连接
	g.UseDB(db)

	// 获取表名
	var tables []string
	if len(dbConfig.Tables) > 0 {
		// 使用指定的表名
		tables = dbConfig.Tables
		fmt.Printf("使用指定的表: %v\n", tables)
	} else {
		// 获取所有表名
		allTables, err := db.Migrator().GetTables()
		if err != nil {
			return fmt.Errorf("获取表列表失败: %v", err)
		}
		tables = allTables
		fmt.Printf("找到 %d 个表: %v\n", len(tables), tables)
	}

	if len(tables) == 0 {
		fmt.Printf("数据库 %s 中没有找到表\n", dbConfig.Name)
		return nil
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

	fmt.Printf("生成的文件位于: %s/\n", dbConfig.OutPath)
	return nil
}
