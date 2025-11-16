-- 添加房间密码字段到game_rooms表

-- 如果password字段不存在，则添加
ALTER TABLE `game_rooms` 
ADD COLUMN IF NOT EXISTS `password` VARCHAR(255) DEFAULT '' COMMENT '房间密码' AFTER `status`,
ADD COLUMN IF NOT EXISTS `has_password` TINYINT(1) DEFAULT 0 COMMENT '是否有密码' AFTER `password`;


