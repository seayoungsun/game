-- 创建管理后台RBAC相关表

-- 管理员表
CREATE TABLE IF NOT EXISTS `admins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(50) NOT NULL COMMENT '管理员用户名',
  `password` VARCHAR(255) NOT NULL COMMENT '密码(加密后)',
  `nickname` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '昵称',
  `email` VARCHAR(100) DEFAULT '' COMMENT '邮箱',
  `avatar` VARCHAR(255) DEFAULT '' COMMENT '头像',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1正常,2禁用',
  `last_login_at` BIGINT DEFAULT 0 COMMENT '最后登录时间',
  `last_login_ip` VARCHAR(50) DEFAULT '' COMMENT '最后登录IP',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  `deleted_at` BIGINT DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_status` (`status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员表';

-- 角色表
CREATE TABLE IF NOT EXISTS `admin_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_code` VARCHAR(50) NOT NULL COMMENT '角色代码',
  `role_name` VARCHAR(50) NOT NULL COMMENT '角色名称',
  `description` VARCHAR(255) DEFAULT '' COMMENT '角色描述',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1启用,2禁用',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_code` (`role_code`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员角色表';

-- 权限表
CREATE TABLE IF NOT EXISTS `admin_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `permission_code` VARCHAR(100) NOT NULL COMMENT '权限代码',
  `permission_name` VARCHAR(100) NOT NULL COMMENT '权限名称',
  `resource` VARCHAR(50) NOT NULL COMMENT '资源类型',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型',
  `parent_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '父权限ID',
  `sort_order` INT DEFAULT 0 COMMENT '排序',
  `description` VARCHAR(255) DEFAULT '' COMMENT '权限描述',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_permission_code` (`permission_code`),
  KEY `idx_resource` (`resource`),
  KEY `idx_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限表';

-- 管理员角色关联表
CREATE TABLE IF NOT EXISTS `admin_role_relations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT UNSIGNED NOT NULL COMMENT '管理员ID',
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_role` (`admin_id`, `role_id`),
  KEY `idx_admin` (`admin_id`),
  KEY `idx_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员角色关联表';

-- 角色权限关联表
CREATE TABLE IF NOT EXISTS `role_permission_relations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  `permission_id` BIGINT UNSIGNED NOT NULL COMMENT '权限ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  KEY `idx_role` (`role_id`),
  KEY `idx_permission` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色权限关联表';

-- 初始化默认角色
INSERT INTO `admin_roles` (`role_code`, `role_name`, `description`, `status`, `created_at`, `updated_at`) VALUES
('super_admin', '超级管理员', '拥有所有权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('admin', '管理员', '拥有除系统管理外的所有权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('operator', '运营', '用户管理和订单查看权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('auditor', '审核员', '订单审核权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE `updated_at`=UNIX_TIMESTAMP();

-- 初始化默认权限
INSERT INTO `admin_permissions` (`permission_code`, `permission_name`, `resource`, `action`, `parent_id`, `sort_order`, `description`, `created_at`) VALUES
-- 仪表盘
('admin:dashboard:view', '查看仪表盘', 'dashboard', 'view', 0, 1, '查看管理后台仪表盘', UNIX_TIMESTAMP()),
-- 用户管理
('admin:users:list', '查看用户列表', 'users', 'list', 0, 10, '查看前端用户列表', UNIX_TIMESTAMP()),
('admin:users:detail', '查看用户详情', 'users', 'detail', 0, 11, '查看前端用户详情', UNIX_TIMESTAMP()),
('admin:users:update', '更新用户信息', 'users', 'update', 0, 12, '更新前端用户信息', UNIX_TIMESTAMP()),
('admin:users:ban', '封禁用户', 'users', 'ban', 0, 13, '封禁或解封前端用户', UNIX_TIMESTAMP()),
-- 充值订单
('admin:recharge_orders:list', '查看充值订单', 'recharge_orders', 'list', 0, 20, '查看充值订单列表', UNIX_TIMESTAMP()),
('admin:recharge_orders:detail', '查看充值订单详情', 'recharge_orders', 'detail', 0, 21, '查看充值订单详情', UNIX_TIMESTAMP()),
-- 提现订单
('admin:withdraw_orders:list', '查看提现订单', 'withdraw_orders', 'list', 0, 30, '查看提现订单列表', UNIX_TIMESTAMP()),
('admin:withdraw_orders:detail', '查看提现订单详情', 'withdraw_orders', 'detail', 0, 31, '查看提现订单详情', UNIX_TIMESTAMP()),
('admin:withdraw_orders:audit', '审核提现订单', 'withdraw_orders', 'audit', 0, 32, '审核提现订单', UNIX_TIMESTAMP()),
-- 充值地址
('admin:deposit_addresses:list', '查看充值地址', 'deposit_addresses', 'list', 0, 40, '查看充值地址列表', UNIX_TIMESTAMP()),
-- 支付管理
('admin:payments:collect', 'USDT归集', 'payments', 'collect', 0, 50, '执行USDT归集', UNIX_TIMESTAMP()),
('admin:payments:batch_collect', '批量归集', 'payments', 'batch_collect', 0, 51, '批量执行USDT归集', UNIX_TIMESTAMP()),
-- 系统管理
('admin:roles:list', '查看角色列表', 'roles', 'list', 0, 60, '查看角色列表', UNIX_TIMESTAMP()),
('admin:roles:create', '创建角色', 'roles', 'create', 0, 61, '创建新角色', UNIX_TIMESTAMP()),
('admin:roles:update', '更新角色', 'roles', 'update', 0, 62, '更新角色信息', UNIX_TIMESTAMP()),
('admin:roles:delete', '删除角色', 'roles', 'delete', 0, 63, '删除角色', UNIX_TIMESTAMP()),
('admin:roles:assign_permission', '分配权限', 'roles', 'assign_permission', 0, 64, '为角色分配权限', UNIX_TIMESTAMP()),
('admin:admins:list', '查看管理员列表', 'admins', 'list', 0, 70, '查看管理员列表', UNIX_TIMESTAMP()),
('admin:admins:create', '创建管理员', 'admins', 'create', 0, 71, '创建新管理员', UNIX_TIMESTAMP()),
('admin:admins:update', '更新管理员', 'admins', 'update', 0, 72, '更新管理员信息', UNIX_TIMESTAMP()),
('admin:admins:delete', '删除管理员', 'admins', 'delete', 0, 73, '删除管理员', UNIX_TIMESTAMP()),
('admin:admins:assign_role', '分配角色', 'admins', 'assign_role', 0, 74, '为管理员分配角色', UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE `description`=VALUES(`description`);

-- 注意：默认管理员密码需要在运行迁移后通过 init_admin.go 脚本设置
-- 或者手动执行以下SQL（密码是 admin123 的 bcrypt 哈希值）

-- 为默认超级管理员分配超级管理员角色（如果管理员已存在）
INSERT INTO `admin_role_relations` (`admin_id`, `role_id`, `created_at`) 
SELECT a.id, r.id, UNIX_TIMESTAMP()
FROM `admins` a, `admin_roles` r
WHERE a.username = 'admin' AND r.role_code = 'super_admin'
ON DUPLICATE KEY UPDATE `created_at`=VALUES(`created_at`);

-- 为超级管理员角色分配所有权限
INSERT INTO `role_permission_relations` (`role_id`, `permission_id`, `created_at`)
SELECT r.id, p.id, UNIX_TIMESTAMP()
FROM `admin_roles` r, `admin_permissions` p
WHERE r.role_code = 'super_admin'
ON DUPLICATE KEY UPDATE `created_at`=VALUES(`created_at`);

