package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/database"
)

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化数据库
	db, err := database.InitMySQL(cfg)
	if err != nil {
		panic(fmt.Sprintf("初始化数据库失败: %v", err))
	}
	defer database.Close()

	// 生成默认管理员密码的哈希值（admin123）
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("加密密码失败: %v", err))
	}

	// 更新或创建默认管理员
	sql := `
		INSERT INTO admins (username, password, nickname, email, status, created_at, updated_at)
		VALUES ('admin', ?, '超级管理员', 'admin@example.com', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
		ON DUPLICATE KEY UPDATE password = VALUES(password), updated_at = UNIX_TIMESTAMP()
	`

	if err := db.Exec(sql, string(hashedPassword)).Error; err != nil {
		panic(fmt.Sprintf("创建默认管理员失败: %v", err))
	}

	// 为管理员分配超级管理员角色
	assignRoleSQL := `
		INSERT INTO admin_role_relations (admin_id, role_id, created_at)
		SELECT a.id, r.id, UNIX_TIMESTAMP()
		FROM admins a, admin_roles r
		WHERE a.username = 'admin' AND r.role_code = 'super_admin'
		ON DUPLICATE KEY UPDATE created_at = UNIX_TIMESTAMP()
	`

	if err := db.Exec(assignRoleSQL).Error; err != nil {
		fmt.Printf("⚠️  分配角色失败（可能角色不存在）: %v\n", err)
		fmt.Println("   请先执行数据库迁移创建角色和权限表")
	} else {
		fmt.Println("✅ 已为管理员分配超级管理员角色")
	}

	fmt.Println("✅ 默认管理员创建成功！")
	fmt.Println("   用户名: admin")
	fmt.Println("   密码: admin123")
	fmt.Println("   角色: 超级管理员（拥有所有权限）")
	fmt.Println("   ⚠️  请首次登录后立即修改密码！")
	os.Exit(0)
}
