-- 初始化数据库表结构

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `uid` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `phone` VARCHAR(20) NOT NULL COMMENT '手机号',
  `nickname` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '昵称',
  `avatar` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '余额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1正常,2封禁',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid` (`uid`),
  UNIQUE KEY `uk_phone` (`phone`),
  KEY `idx_created` (`created_at`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 用户钱包表
CREATE TABLE IF NOT EXISTS `user_wallets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '余额',
  `frozen` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '冻结金额',
  `total_in` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '累计充值',
  `total_out` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '累计提现',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户钱包';

-- 用户登录记录表
CREATE TABLE IF NOT EXISTS `user_logins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `ip` VARCHAR(50) NOT NULL DEFAULT '' COMMENT 'IP地址',
  `device` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '设备信息',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户登录记录';

-- 游戏房间表
CREATE TABLE IF NOT EXISTS `game_rooms` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `game_type` VARCHAR(20) NOT NULL COMMENT '游戏类型:texas/bull/running',
  `room_type` VARCHAR(20) NOT NULL DEFAULT 'quick' COMMENT '房间类型:quick/middle/high',
  `base_bet` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '底注',
  `max_players` INT NOT NULL DEFAULT 4 COMMENT '最大人数',
  `current_players` INT NOT NULL DEFAULT 0 COMMENT '当前人数',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1等待,2游戏中,3已结束',
  `players` JSON COMMENT '玩家列表',
  `creator_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建者ID',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_room_id` (`room_id`),
  KEY `idx_game_type` (`game_type`),
  KEY `idx_status` (`status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏房间';

-- 游戏对局记录表
CREATE TABLE IF NOT EXISTS `game_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `game_type` VARCHAR(20) NOT NULL COMMENT '游戏类型',
  `players` JSON COMMENT '玩家列表',
  `result` JSON COMMENT '结算结果',
  `start_time` TIMESTAMP NOT NULL COMMENT '开始时间',
  `end_time` TIMESTAMP NOT NULL COMMENT '结束时间',
  `duration` INT NOT NULL DEFAULT 0 COMMENT '时长(秒)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_room_id` (`room_id`),
  KEY `idx_game_type` (`game_type`, `start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏对局记录';

-- 游戏玩家关联表
CREATE TABLE IF NOT EXISTS `game_players` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id` VARCHAR(50) NOT NULL COMMENT '房间ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `position` INT NOT NULL DEFAULT 0 COMMENT '位置',
  `balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '本局余额变化',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_room_id` (`room_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='游戏玩家关联';

-- 交易订单表
CREATE TABLE IF NOT EXISTS `transactions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `type` VARCHAR(20) NOT NULL COMMENT '类型:recharge/withdraw/game',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待处理,2成功,3失败',
  `channel` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '支付渠道:alipay/wechat',
  `channel_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '第三方订单号',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_type` (`type`, `status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易订单';

-- 充值订单表
CREATE TABLE IF NOT EXISTS `recharge_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '充值金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待支付,2已支付,3已取消',
  `channel` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '支付渠道',
  `channel_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '第三方订单号',
  `paid_at` TIMESTAMP NULL DEFAULT NULL COMMENT '支付时间',
  `expire_at` TIMESTAMP NOT NULL COMMENT '过期时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_expire` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='充值订单';

-- 提现订单表
CREATE TABLE IF NOT EXISTS `withdraw_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `amount` DECIMAL(10,2) NOT NULL COMMENT '提现金额',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1待审核,2已通过,3已拒绝',
  `bank_card` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '银行卡号',
  `bank_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '银行名称',
  `real_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `audit_at` TIMESTAMP NULL DEFAULT NULL COMMENT '审核时间',
  `auditor_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '审核员ID',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='提现订单';


