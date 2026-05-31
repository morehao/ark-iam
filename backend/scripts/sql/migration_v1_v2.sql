-- ============================================
-- Application 模型拆分迁移脚本
-- 将旧 application 表拆分为 application、tenant_application、oauth_client
-- ============================================

-- 1. 备份旧表
RENAME TABLE `application` TO `application_old`;

-- 2. 创建新 application 表（全局应用定义）
CREATE TABLE `application` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '应用ID',
    `code`            VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用编码',
    `name`            VARCHAR(128) NOT NULL DEFAULT '' COMMENT '应用名称',
    `description`     TEXT DEFAULT NULL COMMENT '应用描述',
    `logo_url`        VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用logo',
    `homepage_url`    VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用主页',
    `type`            VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '应用类型',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `sort`            INT NOT NULL DEFAULT 0 COMMENT '排序',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用定义表';

-- 3. 从旧表提取去重应用定义到新 application 表
INSERT INTO `application` (`code`, `name`, `type`, `status`, `sort`, `created_at`, `updated_at`, `created_by`)
SELECT DISTINCT
    LOWER(REPLACE(`name`, ' ', '_')) AS `code`,
    `name`, `type`, `status`, 0 AS `sort`,
    NOW(), NOW(), 0
FROM `application_old`
WHERE `deleted_at` IS NULL;

-- 4. 创建租户订阅
INSERT INTO `tenant_application` (`tenant_id`, `application_id`, `status`, `created_at`, `updated_at`, `created_by`)
SELECT DISTINCT a.`tenant_id`, ad.`id`, 'enable', NOW(), NOW(), 0
FROM `application_old` a
JOIN `application` ad ON ad.`name` = a.`name`
WHERE a.`deleted_at` IS NULL;

-- 5. 迁移 OIDC 客户端到 oauth_client
INSERT INTO `oauth_client` (
    `tenant_id`, `application_id`, `client_id`, `name`,
    `redirect_uris`, `post_logout_redirect_uris`, `grant_types`, `response_types`,
    `token_endpoint_auth_method`, `allowed_origins`, `require_pkce`, `require_auth_time`,
    `default_scopes`, `access_token_ttl`, `refresh_token_ttl`, `type`, `is_third_party`, `status`,
    `created_at`, `updated_at`, `created_by`, `updated_by`
)
SELECT
    a.`tenant_id`, ad.`id`, a.`client_id`, a.`name`,
    a.`redirect_uris`, a.`post_logout_redirect_uris`, a.`grant_types`, a.`response_types`,
    a.`token_endpoint_auth_method`, a.`allowed_origins`, a.`require_pkce`, a.`require_auth_time`,
    a.`default_scopes`, a.`access_token_ttl`, a.`refresh_token_ttl`, a.`type`, a.`is_third_party`, a.`status`,
    a.`created_at`, a.`updated_at`, a.`created_by`, a.`updated_by`
FROM `application_old` a
JOIN `application` ad ON ad.`name` = a.`name`
WHERE a.`deleted_at` IS NULL;

-- 6. 迁移密钥到 oauth_client_secret
INSERT INTO `oauth_client_secret` (`oauth_client_id`, `name`, `value_hash`, `value_prefix`, `expired_at`, `revoked_at`, `created_at`, `updated_at`, `created_by`, `updated_by`)
SELECT oc.`id`, s.`name`, s.`value_hash`, s.`value_prefix`, s.`expired_at`, s.`revoked_at`, s.`created_at`, s.`updated_at`, s.`created_by`, s.`updated_by`
FROM `application_secret` s
JOIN `oauth_client` oc ON oc.`client_id` = (
    SELECT a.`client_id` FROM `application_old` a WHERE a.`id` = s.`application_id`
)
WHERE s.`deleted_at` IS NULL;

-- 7. 迁移 application_role 数据 — 更新 role.application_id
-- 此时 application_role.application_id 指向旧 application.id，需更新为新的 application.id
UPDATE `application_role` ar
JOIN `application_old` ao ON ao.`id` = ar.`application_id`
JOIN `application` ad ON ad.`name` = ao.`name`
SET ar.`application_id` = ad.`id`
WHERE ar.`deleted_at` IS NULL;

-- 这一步需要在角色表先添加 application_id 列之后再做
-- ALTER TABLE `role` ADD COLUMN `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `tenant_id`;
-- 然后将 application_role 中的数据更新到 role

-- 8. menu 表 — 移除 tenant_id，添加 application_id
ALTER TABLE `menu` DROP COLUMN `tenant_id`;
ALTER TABLE `menu` ADD COLUMN `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id' AFTER `id`;
CREATE INDEX `idx_application_id` ON `menu` (`application_id`);

-- 9. role 表 — 添加 application_id 列
ALTER TABLE `role` ADD COLUMN `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id' AFTER `tenant_id`;
CREATE INDEX `idx_application_id` ON `role` (`application_id`);

-- 10. refresh_token 表 — 重命名列
ALTER TABLE `refresh_token` CHANGE COLUMN `application_id` `oauth_client_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'OIDC客户端ID';

-- 11. 清理旧表
DROP TABLE IF EXISTS `application_secret`;
DROP TABLE IF EXISTS `application_role`;
DROP TABLE IF EXISTS `application_old`;
