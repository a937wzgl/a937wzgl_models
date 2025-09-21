package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("测试数据库连接...")

	db, err := gorm.Open(mysql.Open("root:root123@tcp(127.0.0.1:3306)/fish?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	fmt.Println("连接成功！")

	// 测试查询
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'PROCEDURE'", "fish").Row().Scan(&count)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	fmt.Printf("数据库 fish 中有 %d 个存储过程\n", count)
}
