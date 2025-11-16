package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kaifa/game-platform/internal/config"
)

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("数据库连接失败: %v", err))
	}

	fmt.Println("数据库连接成功，开始执行迁移...")

	// 获取迁移文件目录
	migrationsDir := filepath.Join("../../migrations")

	// 读取所有SQL文件
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		panic(fmt.Sprintf("读取迁移目录失败: %v", err))
	}

	// 过滤并排序SQL文件
	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			// 跳过旧的单文件迁移
			if file.Name() == "add_password_field.sql" {
				continue
			}
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	// 按文件名排序（确保按顺序执行）
	sort.Strings(sqlFiles)

	// 执行每个迁移文件
	for _, fileName := range sqlFiles {
		migrationFile := filepath.Join(migrationsDir, fileName)
		fmt.Printf("\n执行迁移文件: %s\n", fileName)

		sqlBytes, err := os.ReadFile(migrationFile)
		if err != nil {
			fmt.Printf("⚠️  读取迁移文件失败: %v\n", err)
			continue
		}

		// 分割SQL语句
		content := string(sqlBytes)

		// 移除整行注释（以 -- 开头且不在字符串内的行）
		lines := strings.Split(content, "\n")
		var cleanLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// 保留非空且不是整行注释的内容
			if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
				cleanLines = append(cleanLines, line)
			}
		}
		content = strings.Join(cleanLines, "\n")

		// 按分号分割SQL语句
		// 注意：最后一个元素可能是空的（如果文件以分号结尾）
		sqlStatements := strings.Split(content, ";")

		// 调试：显示分割后的语句数量
		fmt.Printf("  检测到 %d 个SQL语句段\n", len(sqlStatements))

		// 执行SQL
		successCount := 0
		errorCount := 0
		skippedCount := 0
		for i, statement := range sqlStatements {
			statement = strings.TrimSpace(statement)
			// 移除语句内的行注释（-- 之后的内容，但不在字符串内）
			// 简单的处理：查找不在引号内的 --
			var cleanedStatement strings.Builder
			inSingleQuote := false
			inDoubleQuote := false
			for j, r := range statement {
				if r == '\'' && (j == 0 || statement[j-1] != '\\') {
					inSingleQuote = !inSingleQuote
					cleanedStatement.WriteRune(r)
				} else if r == '"' && (j == 0 || statement[j-1] != '\\') {
					inDoubleQuote = !inDoubleQuote
					cleanedStatement.WriteRune(r)
				} else if r == '-' && j+1 < len(statement) && statement[j+1] == '-' && !inSingleQuote && !inDoubleQuote {
					// 找到注释开始，停止处理
					break
				} else {
					cleanedStatement.WriteRune(r)
				}
			}
			statement = strings.TrimSpace(cleanedStatement.String())

			// 跳过空语句
			if statement == "" {
				skippedCount++
				continue
			}

			// 调试：显示要执行的SQL语句（仅前100字符）
			preview := statement
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Printf("  [%d] 执行: %s\n", i+1, preview)

			// 执行SQL
			if _, err := db.Exec(statement); err != nil {
				// 对于某些错误（如表已存在），只显示警告
				errMsg := err.Error()
				if strings.Contains(errMsg, "already exists") ||
					strings.Contains(errMsg, "Duplicate column") ||
					strings.Contains(errMsg, "Duplicate key") ||
					strings.Contains(errMsg, "Duplicate entry") {
					skippedCount++
					// 只显示前几个跳过的消息，避免输出过多
					if skippedCount <= 3 {
						fmt.Printf("  ℹ️  跳过（已存在）\n")
					}
				} else {
					fmt.Printf("  ❌ 执行失败（第%d条）: %v\n", i+1, err)
					// 显示前150个字符的SQL
					preview := statement
					if len(preview) > 150 {
						preview = preview[:150] + "..."
					}
					fmt.Printf("     SQL: %s\n", preview)
					errorCount++
				}
			} else {
				successCount++
				// 成功时显示表名（如果是CREATE TABLE）
				if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(statement)), "CREATE TABLE") {
					// 提取表名
					parts := strings.Fields(statement)
					for j, part := range parts {
						if strings.ToUpper(part) == "TABLE" && j+1 < len(parts) {
							tableName := strings.Trim(parts[j+1], "`")
							fmt.Printf("      ✅ 创建表: %s\n", tableName)
							break
						}
					}
				}
			}
		}

		if skippedCount > 3 {
			fmt.Printf("  ℹ️  共跳过 %d 条（已存在）\n", skippedCount)
		}

		if successCount > 0 || errorCount == 0 {
			fmt.Printf("  ✅ 成功执行 %d 条SQL语句\n", successCount)
		}
		if errorCount > 0 {
			fmt.Printf("  ⚠️  有 %d 条SQL语句执行失败\n", errorCount)
		}
	}

	fmt.Println("\n✅ 所有迁移完成！")
}
