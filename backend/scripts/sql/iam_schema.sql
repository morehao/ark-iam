CREATE DATABASE iam
CHARACTER SET utf8mb4
COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `tenant`
(
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '租户ID',
    `name`          VARCHAR(128) NOT NULL DEFAULT '' COMMENT '租户名称',
    `db_user`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '数据库用户',
    `is_suspended`  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否挂起',
    `tag`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '标签',
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户表';

CREATE TABLE `system`
(
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `key`           VARCHAR(256) NOT NULL DEFAULT '' COMMENT '配置键',
    `value`         JSON NOT NULL DEFAULT ('{}') COMMENT '配置值',
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY             `idx_tenant_id` (`tenant_id`),
    KEY             `idx_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='系统配置表';

CREATE TABLE `user`
(
    `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    `tenant_id`             BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `username`              VARCHAR(128) NOT NULL DEFAULT '' COMMENT '用户名',
    `primary_email`         VARCHAR(128) NOT NULL DEFAULT '' COMMENT '主要邮箱',
    `primary_phone`         VARCHAR(128) NOT NULL DEFAULT '' COMMENT '主要手机号',
    `password_encrypted`    VARCHAR(256) NOT NULL DEFAULT '' COMMENT '加密密码',
    `password_method`       VARCHAR(32) NOT NULL DEFAULT '' COMMENT '密码加密方式',
    `name`                  VARCHAR(128) NOT NULL DEFAULT '' COMMENT '姓名',
    `avatar`                VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '头像URL',
    `profile`               JSON NOT NULL DEFAULT ('{}') COMMENT '配置信息',
    `application_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `identities`            JSON NOT NULL DEFAULT ('{}') COMMENT '第三方身份',
    `custom_data`           JSON NOT NULL DEFAULT ('{}') COMMENT '自定义数据',
    `is_suspended`          TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否挂起',
    `last_sign_in_at`       DATETIME DEFAULT NULL COMMENT '最后登录时间',
    `created_at`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`            DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`            BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`            BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`            BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY                     `idx_tenant_id` (`tenant_id`),
    KEY                     `idx_tenant_username` (`tenant_id`, `username`),
    KEY                     `idx_tenant_email` (`tenant_id`, `primary_email`),
    KEY                     `idx_tenant_phone` (`tenant_id`, `primary_phone`),
    KEY                     `idx_tenant_name` (`tenant_id`, `name`),
    KEY                     `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户表';

CREATE TABLE `user_login_log`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `login_ip`       VARCHAR(32) DEFAULT NULL COMMENT '登录IP地址',
    `user_agent`     VARCHAR(512) DEFAULT NULL COMMENT '用户代理信息',
    `login_time`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_login_time` (`login_time`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户登录日志表';

CREATE TABLE `user_identity`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `issuer`         VARCHAR(256) NOT NULL DEFAULT '' COMMENT '身份提供商',
    `identity_id`    VARCHAR(128) NOT NULL DEFAULT '' COMMENT '第三方用户ID',
    `detail`         JSON NOT NULL DEFAULT ('{}') COMMENT '详细信息',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_issuer_identity` (`tenant_id`, `issuer`, `identity_id`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户身份表';

CREATE TABLE `department`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '部门ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `parent_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父部门ID',
    `name`           VARCHAR(128) NOT NULL DEFAULT '' COMMENT '部门名称',
    `code`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '部门编码',
    `sort`           INT NOT NULL DEFAULT 0 COMMENT '排序',
    `leader_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '部门负责人用户ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_tenant_parent_id` (`tenant_id`, `parent_id`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门表';

CREATE TABLE `user_department_relation`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `department_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '部门ID',
    `is_primary`     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否主部门',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_department_id` (`department_id`),
    KEY              `idx_tenant_user_dept` (`tenant_id`, `user_id`, `department_id`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户部门关系表';

CREATE TABLE `application`
(
    `id`                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '应用ID',
    `tenant_id`            BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`                 VARCHAR(256) NOT NULL DEFAULT '' COMMENT '应用名称',
    `secret`               VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用密钥',
    `description`          TEXT DEFAULT NULL COMMENT '应用描述',
    `type`                 VARCHAR(32) NOT NULL DEFAULT '' COMMENT '应用类型',
    `oidc_client_metadata` JSON NOT NULL DEFAULT ('{}') COMMENT 'OIDC客户端配置',
    `custom_client_metadata` JSON NOT NULL DEFAULT ('{}') COMMENT '自定义客户端配置',
    `is_third_party`       TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否第三方应用',
    `created_at`           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`           DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY                    `idx_tenant_id` (`tenant_id`),
    KEY                    `idx_tenant_type` (`tenant_id`, `type`),
    KEY                    `idx_tenant_is_third_party` (`tenant_id`, `is_third_party`),
    KEY                    `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用表';

CREATE TABLE `application_secret`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `name`            VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥名称',
    `value`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '密钥值',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `expires_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY               `idx_tenant_id` (`tenant_id`),
    KEY               `idx_application_id` (`application_id`),
    KEY               `idx_tenant_app_name` (`tenant_id`, `application_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用密钥表';

CREATE TABLE `resource`
(
    `id`                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '资源ID',
    `tenant_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`                VARCHAR(256) NOT NULL DEFAULT '' COMMENT '资源名称',
    `indicator`           VARCHAR(512) NOT NULL DEFAULT '' COMMENT '资源标识符',
    `is_default`          TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认',
    `access_token_ttl`    BIGINT NOT NULL DEFAULT 3600 COMMENT '访问令牌TTL',
    `created_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`          DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY                   `idx_tenant_id` (`tenant_id`),
    KEY                   `idx_tenant_indicator` (`tenant_id`, `indicator`),
    KEY                   `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='资源表';

CREATE TABLE `scope`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '权限ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `resource_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '资源ID',
    `name`           VARCHAR(256) NOT NULL DEFAULT '' COMMENT '权限名称',
    `description`    TEXT DEFAULT NULL COMMENT '权限描述',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_resource_id` (`resource_id`),
    KEY              `idx_tenant_resource_name` (`tenant_id`, `resource_id`, `name`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='权限范围表';

CREATE TABLE `role`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`           VARCHAR(128) NOT NULL DEFAULT '' COMMENT '角色名称',
    `code`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色编码',
    `description`   VARCHAR(256) NOT NULL DEFAULT '' COMMENT '角色描述',
    `type`           VARCHAR(32) NOT NULL DEFAULT 'User' COMMENT '角色类型',
    `is_default`    TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认角色',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_tenant_name` (`tenant_id`, `name`),
    KEY              `idx_tenant_code` (`tenant_id`, `code`),
    KEY              `idx_tenant_type` (`tenant_id`, `type`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色表';

CREATE TABLE `role_scope`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `role_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色ID',
    `scope_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '权限ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_role_id` (`role_id`),
    KEY              `idx_scope_id` (`scope_id`),
    KEY              `idx_tenant_role_scope` (`tenant_id`, `role_id`, `scope_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色权限关联表';

CREATE TABLE `user_role`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `role_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_role_id` (`role_id`),
    KEY              `idx_tenant_user_role` (`tenant_id`, `user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户角色关联表';

CREATE TABLE `application_role`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `role_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_application_id` (`application_id`),
    KEY              `idx_role_id` (`role_id`),
    KEY              `idx_tenant_app_role` (`tenant_id`, `application_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用角色关联表';

CREATE TABLE `organization`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '组织ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`           VARCHAR(128) NOT NULL DEFAULT '' COMMENT '组织名称',
    `description`    VARCHAR(256) NOT NULL DEFAULT '' COMMENT '组织描述',
    `custom_data`    JSON NOT NULL DEFAULT ('{}') COMMENT '自定义数据',
    `is_mfa_required` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要MFA',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织表';

CREATE TABLE `organization_role`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '组织角色ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `organization_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '组织ID',
    `name`           VARCHAR(128) NOT NULL DEFAULT '' COMMENT '角色名称',
    `description`    VARCHAR(256) NOT NULL DEFAULT '' COMMENT '角色描述',
    `type`           VARCHAR(32) NOT NULL DEFAULT 'User' COMMENT '角色类型',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_organization_id` (`organization_id`),
    KEY              `idx_tenant_org_name` (`tenant_id`, `organization_id`, `name`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织角色表';

CREATE TABLE `organization_user_relation`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `organization_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '组织ID',
    `user_id`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY               `idx_tenant_id` (`tenant_id`),
    KEY               `idx_organization_id` (`organization_id`),
    KEY               `idx_user_id` (`user_id`),
    KEY               `idx_tenant_org_user` (`tenant_id`, `organization_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织用户关系表';

CREATE TABLE `organization_role_user_relation`
(
    `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `organization_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '组织ID',
    `organization_role_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '组织角色ID',
    `user_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`        DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY                 `idx_tenant_id` (`tenant_id`),
    KEY                 `idx_tenant_org_user` (`tenant_id`, `organization_id`, `user_id`),
    KEY                 `idx_tenant_org_role_user` (`tenant_id`, `organization_id`, `organization_role_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织角色用户关系表';

CREATE TABLE `menu`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `parent_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父菜单ID',
    `name`           VARCHAR(128) NOT NULL DEFAULT '' COMMENT '菜单名称',
    `code`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '菜单编码',
    `path`           VARCHAR(512) NOT NULL DEFAULT '' COMMENT '菜单路径',
    `icon`           VARCHAR(256) NOT NULL DEFAULT '' COMMENT '菜单图标',
    `sort`           INT NOT NULL DEFAULT 0 COMMENT '排序',
    `type`           VARCHAR(32) NOT NULL DEFAULT '' COMMENT '菜单类型',
    `component`      VARCHAR(256) NOT NULL DEFAULT '' COMMENT '组件路径',
    `redirect`       VARCHAR(512) NOT NULL DEFAULT '' COMMENT '重定向路径',
    `hidden`         TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否隐藏',
    `external_link`  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否外链',
    `keep_alive`     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否缓存',
    `permission`     VARCHAR(128) NOT NULL DEFAULT '' COMMENT '权限标识',
    `status`         VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_tenant_parent_id` (`tenant_id`, `parent_id`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单表';

CREATE TABLE `role_menu`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `role_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '角色ID',
    `menu_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '菜单ID',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_role_id` (`role_id`),
    KEY              `idx_menu_id` (`menu_id`),
    KEY              `idx_tenant_role_menu` (`tenant_id`, `role_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色菜单关联表';

CREATE TABLE `connector`
(
    `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '连接器ID',
    `tenant_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `sync_profile`       TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否同步资料',
    `enable_token_storage` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用令牌存储',
    `connector_id`       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '连接器ID',
    `config`             JSON NOT NULL DEFAULT ('{}') COMMENT '连接器配置',
    `metadata`           JSON NOT NULL DEFAULT ('{}') COMMENT '元数据',
    `created_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`         DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY                  `idx_tenant_id` (`tenant_id`),
    KEY                  `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='连接器表';

CREATE TABLE `sso_connector`
(
    `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'SSO连接器ID',
    `tenant_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `provider_name`      VARCHAR(128) NOT NULL DEFAULT '' COMMENT '提供商名称',
    `connector_name`     VARCHAR(128) NOT NULL DEFAULT '' COMMENT '连接器名称',
    `config`             JSON NOT NULL DEFAULT ('{}') COMMENT '配置',
    `domains`            JSON NOT NULL DEFAULT ('[]') COMMENT '域名列表',
    `branding`           JSON NOT NULL DEFAULT ('{}') COMMENT '品牌配置',
    `sync_profile`       TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否同步资料',
    `enable_token_storage` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用令牌存储',
    `created_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`         DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY                  `idx_tenant_id` (`tenant_id`),
    KEY                  `idx_tenant_connector_name` (`tenant_id`, `connector_name`),
    KEY                  `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='SSO连接器表';

CREATE TABLE `log`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '日志ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `key`            VARCHAR(128) NOT NULL DEFAULT '' COMMENT '日志键',
    `payload`        JSON NOT NULL DEFAULT ('{}') COMMENT '日志内容',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_tenant_key` (`tenant_id`, `key`),
    KEY              `idx_tenant_created_at` (`tenant_id`, `created_at`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='日志表';

CREATE TABLE `refresh_token`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `application_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用ID',
    `token`          VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'token哈希',
    `expires_at`     DATETIME DEFAULT NULL COMMENT '过期时间',
    `revoked_at`     DATETIME DEFAULT NULL COMMENT '撤销时间',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_token` (`token`),
    KEY              `idx_tenant_user` (`tenant_id`, `user_id`),
    KEY              `idx_expires_at` (`expires_at`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='刷新令牌表';
