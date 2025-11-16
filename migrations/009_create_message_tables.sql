-- 系统公告表
CREATE TABLE IF NOT EXISTS `announcements` (
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

-- 用户消息表
CREATE TABLE IF NOT EXISTS `user_messages` (
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











