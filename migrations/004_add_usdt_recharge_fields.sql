-- 添加USDT充值相关字段到recharge_orders表

ALTER TABLE `recharge_orders` 
ADD COLUMN `chain_type` VARCHAR(20) DEFAULT '' COMMENT '链类型:trc20/erc20' AFTER `channel_id`,
ADD COLUMN `deposit_addr` VARCHAR(100) DEFAULT '' COMMENT '充值地址' AFTER `chain_type`,
ADD COLUMN `tx_hash` VARCHAR(128) DEFAULT '' COMMENT '交易哈希' AFTER `deposit_addr`,
ADD COLUMN `confirm_count` INT DEFAULT 0 COMMENT '确认次数' AFTER `tx_hash`,
ADD COLUMN `required_conf` INT DEFAULT 12 COMMENT '需要确认次数' AFTER `confirm_count`;

-- 添加索引
ALTER TABLE `recharge_orders` 
ADD INDEX `idx_deposit_addr` (`deposit_addr`),
ADD INDEX `idx_tx_hash` (`tx_hash`);











