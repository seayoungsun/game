-- 添加密码字段到users表

ALTER TABLE `users` ADD COLUMN `password` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '密码(加密后)' AFTER `phone`;










