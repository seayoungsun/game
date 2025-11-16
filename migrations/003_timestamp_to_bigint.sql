-- 将所有时间字段从TIMESTAMP改为BIGINT（存储Unix时间戳）

-- Users表
ALTER TABLE `users` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';
ALTER TABLE `users` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';

-- UserWallets表
ALTER TABLE `user_wallets` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';

-- UserLogins表
ALTER TABLE `user_logins` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';

-- GameRooms表
ALTER TABLE `game_rooms` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';
ALTER TABLE `game_rooms` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';

-- GameRecords表
ALTER TABLE `game_records` MODIFY COLUMN `start_time` BIGINT NOT NULL DEFAULT 0 COMMENT '开始时间(Unix时间戳)';
ALTER TABLE `game_records` MODIFY COLUMN `end_time` BIGINT NOT NULL DEFAULT 0 COMMENT '结束时间(Unix时间戳)';
ALTER TABLE `game_records` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';

-- GamePlayers表
ALTER TABLE `game_players` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';

-- Transactions表
ALTER TABLE `transactions` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';
ALTER TABLE `transactions` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';

-- RechargeOrders表
ALTER TABLE `recharge_orders` MODIFY COLUMN `paid_at` BIGINT NULL DEFAULT NULL COMMENT '支付时间(Unix时间戳)';
ALTER TABLE `recharge_orders` MODIFY COLUMN `expire_at` BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间(Unix时间戳)';
ALTER TABLE `recharge_orders` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';
ALTER TABLE `recharge_orders` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';

-- WithdrawOrders表
ALTER TABLE `withdraw_orders` MODIFY COLUMN `audit_at` BIGINT NULL DEFAULT NULL COMMENT '审核时间(Unix时间戳)';
ALTER TABLE `withdraw_orders` MODIFY COLUMN `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(Unix时间戳)';
ALTER TABLE `withdraw_orders` MODIFY COLUMN `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(Unix时间戳)';










