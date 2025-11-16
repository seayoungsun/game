-- migrations/005_create_user_deposit_addresses.sql

-- 创建用户充值地址表
CREATE TABLE IF NOT EXISTS `user_deposit_addresses` (
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












