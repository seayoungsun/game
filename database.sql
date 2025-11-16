-- ============================================
-- 游戏平台数据库 - 完整版
-- 包含所有表结构和字段
-- MySQL 5.7+ / 8.0+ 兼容
-- ============================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 用户相关表
-- ============================================

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `uid` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `phone` VARCHAR(20) NOT NULL COMMENT '手机号',
  `password` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '密码(加密后)',
  `nickname` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '昵称',
  `avatar` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '余额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1正常,2封禁',
  `role` VARCHAR(20) NOT NULL DEFAULT 'user' COMMENT '用户角色:user普通用户,admin管理员,operator运营',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  `deleted_at` BIGINT DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid` (`uid`),
  UNIQUE KEY `uk_phone` (`phone`),
  KEY `idx_created` (`created_at`),
  KEY `idx_status` (`status`),
  KEY `idx_role` (`role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

DROP TABLE IF EXISTS `user_wallets`;
CREATE TABLE `user_wallets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '余额',
  `frozen` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '冻结金额',
  `total_in` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '累计充值',
  `total_out` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '累计提现',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户钱包';

DROP TABLE IF EXISTS `user_logins`;
CREATE TABLE `user_logins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `ip` VARCHAR(50) NOT NULL DEFAULT '' COMMENT 'IP地址',
  `device` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '设备信息',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户登录记录';

DROP TABLE IF EXISTS `user_deposit_addresses`;
CREATE TABLE `user_deposit_addresses` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `chain_type` VARCHAR(20) NOT NULL COMMENT '链类型:trc20/erc20',
  `address` VARCHAR(100) NOT NULL COMMENT '充值地址',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_chain` (`user_id`, `chain_type`),
  UNIQUE KEY `uk_address` (`address`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_chain_type` (`chain_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户充值地址表';

DROP TABLE IF EXISTS `user_roles`;
CREATE TABLE `user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(20) NOT NULL COMMENT '角色名称',
  `code` VARCHAR(20) NOT NULL UNIQUE COMMENT '角色代码',
  `description` VARCHAR(255) DEFAULT '' COMMENT '角色描述',
  `permissions` JSON COMMENT '权限列表',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色表';

-- ============================================
-- 游戏相关表
-- ============================================

DROP TABLE IF EXISTS `game_rooms`;
CREATE TABLE `game_rooms` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `game_type` VARCHAR(20) NOT NULL COMMENT '游戏类型:texas/bull/running',
  `room_type` VARCHAR(20) NOT NULL DEFAULT 'quick' COMMENT '房间类型:quick/middle/high',
  `base_bet` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '底注',
  `max_players` INT NOT NULL DEFAULT 4 COMMENT '最大人数',
  `current_players` INT NOT NULL DEFAULT 0 COMMENT '当前人数',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1等待,2游戏中,3已结束',
  `password` VARCHAR(255) DEFAULT '' COMMENT '房间密码',
  `has_password` TINYINT(1) DEFAULT 0 COMMENT '是否有密码',
  `players` JSON COMMENT '玩家列表',
  `creator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建者ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_room_id` (`room_id`),
  KEY `idx_game_type` (`game_type`),
  KEY `idx_status` (`status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏房间';

DROP TABLE IF EXISTS `game_records`;
CREATE TABLE `game_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `game_type` VARCHAR(20) NOT NULL COMMENT '游戏类型',
  `players` JSON COMMENT '玩家列表',
  `result` JSON COMMENT '结算结果',
  `start_time` BIGINT NOT NULL DEFAULT 0 COMMENT '开始时间(Unix时间戳)',
  `end_time` BIGINT NOT NULL DEFAULT 0 COMMENT '结束时间(Unix时间戳)',
  `duration` INT NOT NULL DEFAULT 0 COMMENT '时长(秒)',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  KEY `idx_room_id` (`room_id`),
  KEY `idx_game_type` (`game_type`, `start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏对局记录';

DROP TABLE IF EXISTS `game_players`;
CREATE TABLE `game_players` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `position` INT NOT NULL DEFAULT 0 COMMENT '位置',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '本局余额变化',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  KEY `idx_room_id` (`room_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏玩家关联';

-- ============================================
-- 支付相关表
-- ============================================

DROP TABLE IF EXISTS `transactions`;
CREATE TABLE `transactions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `type` VARCHAR(20) NOT NULL COMMENT '类型:recharge/withdraw/game',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待处理,2成功,3失败',
  `channel` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '支付渠道:alipay/wechat',
  `channel_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '第三方订单号',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_type` (`type`, `status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易订单';

DROP TABLE IF EXISTS `recharge_orders`;
CREATE TABLE `recharge_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '充值金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待支付,2已支付,3已取消',
  `channel` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '支付渠道',
  `channel_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '第三方订单号',
  `chain_type` VARCHAR(20) DEFAULT '' COMMENT '链类型:trc20/erc20',
  `deposit_addr` VARCHAR(100) DEFAULT '' COMMENT '充值地址',
  `tx_hash` VARCHAR(128) DEFAULT '' COMMENT '交易哈希',
  `confirm_count` INT DEFAULT 0 COMMENT '确认次数',
  `required_conf` INT DEFAULT 12 COMMENT '需要确认次数',
  `paid_at` BIGINT NULL DEFAULT NULL COMMENT '支付时间(Unix时间戳)',
  `expire_at` BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间(Unix时间戳)',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_expire` (`expire_at`),
  KEY `idx_deposit_addr` (`deposit_addr`),
  KEY `idx_tx_hash` (`tx_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='充值订单';

DROP TABLE IF EXISTS `withdraw_orders`;
CREATE TABLE `withdraw_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '提现金额',
  `fee` DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '手续费',
  `actual_amount` DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '实际到账金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待审核,2已通过,3已拒绝',
  `channel` VARCHAR(20) COMMENT '支付渠道:usdt_trc20/usdt_erc20',
  `chain_type` VARCHAR(20) COMMENT '链类型:trc20/erc20',
  `to_address` VARCHAR(100) COMMENT '提现地址',
  `tx_hash` VARCHAR(128) COMMENT '交易哈希',
  `confirm_count` INT NOT NULL DEFAULT 0 COMMENT '确认次数',
  `bank_card` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '银行卡号',
  `bank_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '银行名称',
  `real_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `audit_at` BIGINT NULL DEFAULT NULL COMMENT '审核时间(Unix时间戳)',
  `auditor_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '审核员ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)',
  `deleted_at` BIGINT DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_to_address` (`to_address`),
  KEY `idx_tx_hash_withdraw` (`tx_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='提现订单';

-- ============================================
-- 管理后台相关表
-- ============================================

DROP TABLE IF EXISTS `admins`;
CREATE TABLE `admins` (
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

DROP TABLE IF EXISTS `admin_roles`;
CREATE TABLE `admin_roles` (
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

DROP TABLE IF EXISTS `admin_permissions`;
CREATE TABLE `admin_permissions` (
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

DROP TABLE IF EXISTS `admin_role_relations`;
CREATE TABLE `admin_role_relations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT UNSIGNED NOT NULL COMMENT '管理员ID',
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_role` (`admin_id`, `role_id`),
  KEY `idx_admin` (`admin_id`),
  KEY `idx_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员角色关联表';

DROP TABLE IF EXISTS `role_permission_relations`;
CREATE TABLE `role_permission_relations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  `permission_id` BIGINT UNSIGNED NOT NULL COMMENT '权限ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  KEY `idx_role` (`role_id`),
  KEY `idx_permission` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色权限关联表';

DROP TABLE IF EXISTS `admin_operation_logs`;
CREATE TABLE `admin_operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT UNSIGNED NOT NULL COMMENT '管理员ID',
  `admin_name` VARCHAR(50) NOT NULL COMMENT '管理员用户名',
  `module` VARCHAR(50) NOT NULL COMMENT '操作模块',
  `action` VARCHAR(50) NOT NULL COMMENT '操作动作',
  `method` VARCHAR(10) COMMENT 'HTTP方法',
  `path` VARCHAR(255) COMMENT '请求路径',
  `ip` VARCHAR(50) COMMENT 'IP地址',
  `user_agent` VARCHAR(255) COMMENT '用户代理',
  `request` TEXT COMMENT '请求参数',
  `response` TEXT COMMENT '响应结果',
  `status` INT NOT NULL DEFAULT 1 COMMENT '状态:1成功,2失败',
  `error_msg` TEXT COMMENT '错误信息',
  `duration` BIGINT COMMENT '耗时(毫秒)',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_admin_id` (`admin_id`),
  KEY `idx_module` (`module`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员操作日志';

DROP TABLE IF EXISTS `system_configs`;
CREATE TABLE `system_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(100) NOT NULL COMMENT '配置键',
  `config_value` TEXT COMMENT '配置值',
  `config_type` VARCHAR(20) NOT NULL DEFAULT 'string' COMMENT '配置类型:string/int/float/bool/json',
  `group_name` VARCHAR(50) NOT NULL DEFAULT 'default' COMMENT '配置分组',
  `description` VARCHAR(255) COMMENT '配置说明',
  `is_public` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否公开',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`),
  KEY `idx_group_name` (`group_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置';

-- ============================================
-- 消息相关表
-- ============================================

DROP TABLE IF EXISTS `announcements`;
CREATE TABLE `announcements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(200) NOT NULL COMMENT '公告标题',
  `content` TEXT NOT NULL COMMENT '公告内容',
  `type` VARCHAR(20) NOT NULL DEFAULT 'info' COMMENT '公告类型:info/warning/error/success',
  `priority` INT NOT NULL DEFAULT 0 COMMENT '优先级:0普通,1重要,2紧急',
  `status` INT NOT NULL DEFAULT 1 COMMENT '状态:1发布,2下架',
  `start_time` BIGINT COMMENT '开始时间',
  `end_time` BIGINT COMMENT '结束时间',
  `target_users` TEXT COMMENT '目标用户:all=全部,user_id1,user_id2=指定用户',
  `created_by` BIGINT UNSIGNED COMMENT '创建人ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_start_time` (`start_time`),
  KEY `idx_end_time` (`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统公告';

DROP TABLE IF EXISTS `user_messages`;
CREATE TABLE `user_messages` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `type` VARCHAR(20) NOT NULL DEFAULT 'info' COMMENT '消息类型:info/warning/error/success/system/order',
  `title` VARCHAR(200) NOT NULL COMMENT '消息标题',
  `content` TEXT NOT NULL COMMENT '消息内容',
  `related_id` VARCHAR(64) COMMENT '关联ID(如订单号)',
  `is_read` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已读',
  `read_at` BIGINT COMMENT '阅读时间',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_is_read` (`is_read`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_type` (`type`),
  KEY `idx_related_id` (`related_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户消息';

-- ============================================
-- 初始化数据
-- ============================================

-- 插入默认用户角色
INSERT INTO `user_roles` (`name`, `code`, `description`, `permissions`, `created_at`, `updated_at`) VALUES
('普通用户', 'user', '普通玩家用户', '[]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('管理员', 'admin', '系统管理员，拥有所有权限', '["*"]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('运营', 'operator', '运营人员，可以管理订单和用户', '["order:read", "order:audit", "user:read"]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE `updated_at` = UNIX_TIMESTAMP();

-- 插入默认管理员角色
INSERT INTO `admin_roles` (`role_code`, `role_name`, `description`, `status`, `created_at`, `updated_at`) VALUES
('super_admin', '超级管理员', '拥有所有权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('admin', '管理员', '拥有除系统管理外的所有权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('operator', '运营', '用户管理和订单查看权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('auditor', '审核员', '订单审核权限', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE `updated_at`=UNIX_TIMESTAMP();

-- 插入默认权限
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

-- 插入默认系统配置
INSERT INTO `system_configs` (`config_key`, `config_value`, `config_type`, `group_name`, `description`, `is_public`, `created_at`, `updated_at`) VALUES
('site_name', '游戏平台', 'string', 'site', '站点名称', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('site_description', '专业的游戏平台', 'string', 'site', '站点描述', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('min_recharge_amount', '10', 'float', 'payment', '最小充值金额', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('max_recharge_amount', '10000', 'float', 'payment', '最大充值金额', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('min_withdraw_amount', '50', 'float', 'payment', '最小提现金额', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('max_withdraw_amount', '5000', 'float', 'payment', '最大提现金额', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('withdraw_fee_rate', '0.001', 'float', 'payment', '提现手续费率', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('game_room_timeout', '300', 'int', 'game', '游戏房间超时时间(秒)', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('max_room_players', '4', 'int', 'game', '房间最大人数', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('maintenance_mode', 'false', 'bool', 'system', '维护模式', 0, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('maintenance_message', '系统维护中，请稍后再试', 'string', 'system', '维护提示信息', 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE `updated_at` = UNIX_TIMESTAMP();

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 说明：
-- 1. 所有时间字段使用BIGINT存储Unix时间戳
-- 2. 默认管理员密码需要通过 init_admin.go 脚本设置
-- 3. 执行后运行: make init-admin 初始化默认管理员
-- ============================================



