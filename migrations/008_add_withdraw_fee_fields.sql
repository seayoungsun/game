-- 添加提现订单手续费字段
ALTER TABLE `withdraw_orders` 
ADD COLUMN `fee` DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '手续费' AFTER `amount`,
ADD COLUMN `actual_amount` DECIMAL(10,2) NOT NULL DEFAULT 0 COMMENT '实际到账金额' AFTER `fee`;

-- 更新现有订单的实际到账金额（假设手续费为0）
UPDATE `withdraw_orders` SET `actual_amount` = `amount` WHERE `actual_amount` = 0;











