-- migrations/006_add_usdt_withdraw_fields.sql

-- 为提现订单表添加USDT相关字段
ALTER TABLE `withdraw_orders`
    ADD COLUMN `channel` VARCHAR(20) COMMENT '支付渠道:usdt_trc20/usdt_erc20' AFTER `status`,
    ADD COLUMN `chain_type` VARCHAR(20) COMMENT '链类型:trc20/erc20' AFTER `channel`,
    ADD COLUMN `to_address` VARCHAR(100) COMMENT '提现地址' AFTER `chain_type`,
    ADD COLUMN `tx_hash` VARCHAR(128) COMMENT '交易哈希' AFTER `to_address`,
    ADD COLUMN `confirm_count` INT NOT NULL DEFAULT 0 COMMENT '确认次数' AFTER `tx_hash`;

-- 为 to_address 和 tx_hash 添加索引
CREATE INDEX idx_to_address ON `withdraw_orders` (`to_address`);
CREATE INDEX idx_tx_hash_withdraw ON `withdraw_orders` (`tx_hash`);











