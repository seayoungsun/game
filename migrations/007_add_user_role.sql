-- 添加用户角色字段
ALTER TABLE `users` 
ADD COLUMN `role` VARCHAR(20) NOT NULL DEFAULT 'user' COMMENT '用户角色:user普通用户,admin管理员,operator运营' AFTER `status`;

-- 添加角色索引
ALTER TABLE `users` ADD INDEX `idx_role` (`role`);

-- 创建角色表（可选，用于更复杂的权限管理）
CREATE TABLE IF NOT EXISTS `user_roles` (
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

-- 插入默认角色
INSERT INTO `user_roles` (`name`, `code`, `description`, `permissions`, `created_at`, `updated_at`) VALUES
('普通用户', 'user', '普通玩家用户', '[]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('管理员', 'admin', '系统管理员，拥有所有权限', '["*"]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
('运营', 'operator', '运营人员，可以管理订单和用户', '["order:read", "order:audit", "user:read"]', UNIX_TIMESTAMP(), UNIX_TIMESTAMP());











