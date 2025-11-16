-- 管理员操作日志表
CREATE TABLE IF NOT EXISTS `admin_operation_logs` (
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

-- 系统配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
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











