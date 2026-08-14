CREATE DATABASE iam
CHARACTER SET utf8mb4
COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `tenant`
(
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '租户ID',
    `code`          VARCHAR(64) NOT NULL DEFAULT '' COMMENT '租户编码',
    `name`          VARCHAR(128) NOT NULL DEFAULT '' COMMENT '租户名称',
    `type`          VARCHAR(32) NOT NULL DEFAULT 'customer' COMMENT '租户类型: customer-客户租户, platform-平台租户',
    `db_user`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '数据库用户',
    `is_suspended`  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否挂起',
    `tag`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '标签',
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    UNIQUE KEY       `uk_code` (`code`)
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

CREATE TABLE `person`
(
    `id`                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自然人ID',
    `username`            VARCHAR(128) DEFAULT NULL COMMENT '全局用户名',
    `primary_email`       VARCHAR(128) DEFAULT NULL COMMENT '主要邮箱',
    `primary_phone`       VARCHAR(128) DEFAULT NULL COMMENT '主要手机号',
    `password_encrypted`  VARCHAR(256) NOT NULL DEFAULT '' COMMENT '加密密码',
    `password_method`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '密码加密方式',
    `name`                VARCHAR(128) NOT NULL DEFAULT '' COMMENT '姓名',
    `avatar`              VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '头像URL',
    `profile`             JSON NOT NULL DEFAULT ('{}') COMMENT '配置信息',
    `custom_data`         JSON NOT NULL DEFAULT ('{}') COMMENT '自定义数据',
    `is_suspended`        TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否挂起',
    `last_sign_in_at`     DATETIME DEFAULT NULL COMMENT '最后登录时间',
    `created_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`          DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    UNIQUE KEY            `uk_username` (`username`),
    UNIQUE KEY            `uk_primary_email` (`primary_email`),
    UNIQUE KEY            `uk_primary_phone` (`primary_phone`),
    KEY                   `idx_name` (`name`),
    KEY                   `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='自然人表';

CREATE TABLE `user`
(
    `id`                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    `tenant_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `person_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '自然人ID',
    `name`                VARCHAR(128) NOT NULL DEFAULT '' COMMENT '租户内姓名',
    `avatar`              VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '租户内头像URL',
    `profile`             JSON NOT NULL DEFAULT ('{}') COMMENT '租户内配置信息',
    `custom_data`         JSON NOT NULL DEFAULT ('{}') COMMENT '租户内自定义数据',
    `is_suspended`        TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否挂起',
    `is_owner`            TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否租户拥有者',
    `joined_at`           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入租户时间',
    `last_sign_in_at`     DATETIME DEFAULT NULL COMMENT '最后登录时间',
    `created_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`          DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    KEY                   `idx_tenant_id` (`tenant_id`),
    KEY                   `idx_person_id` (`person_id`),
    UNIQUE KEY            `uk_tenant_person` (`tenant_id`, `person_id`),
    KEY                   `idx_tenant_name` (`tenant_id`, `name`),
    KEY                   `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户用户表';

CREATE TABLE `user_login_log`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `person_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '自然人ID',
    `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `login_type`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '登录类型',
    `login_ip`       VARCHAR(64) DEFAULT NULL COMMENT '登录IP地址',
    `user_agent`     VARCHAR(512) DEFAULT NULL COMMENT '用户代理信息',
    `login_time`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`     DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY              `idx_person_id` (`person_id`),
    KEY              `idx_tenant_id` (`tenant_id`),
    KEY              `idx_user_id` (`user_id`),
    KEY              `idx_login_time` (`login_time`),
    KEY              `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户登录日志表';

CREATE TABLE `user_identity`
(
    `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `person_id`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '自然人ID',
    `connector_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '连接器ID',
    `provider`          VARCHAR(128) NOT NULL DEFAULT '' COMMENT '身份提供商',
    `issuer`            VARCHAR(256) NOT NULL DEFAULT '' COMMENT '身份签发方',
    `external_subject`  VARCHAR(128) NOT NULL DEFAULT '' COMMENT '外部主体标识',
    `detail`            JSON NOT NULL DEFAULT ('{}') COMMENT '详细信息',
    `last_used_at`      DATETIME DEFAULT NULL COMMENT '最后使用时间',
    `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`        DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY                 `idx_person_id` (`person_id`),
    KEY                 `idx_connector_id` (`connector_id`),
    KEY                 `idx_provider` (`provider`),
    UNIQUE KEY          `uk_issuer_subject` (`issuer`, `external_subject`),
    KEY                 `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='自然人外部身份表';

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

CREATE TABLE `user_department`
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户部门表';

CREATE TABLE `application`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '应用ID',
    `code`            VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用编码',
    `name`            VARCHAR(128) NOT NULL DEFAULT '' COMMENT '应用名称',
    `description`     TEXT DEFAULT NULL COMMENT '应用描述',
    `logo_url`        VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用logo',
    `homepage_url`    VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用主页',
    `type`            VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '应用类型',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `sort`            INT NOT NULL DEFAULT 0 COMMENT '排序',
    `tenant_policy`   JSON NOT NULL DEFAULT ('{}') COMMENT '租户策略',
    `is_system`       TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否系统内置',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY        `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用定义表';

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
    `app_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id',
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
    KEY              `idx_app_id` (`app_id`),
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

CREATE TABLE `organization_user`
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织用户表';

CREATE TABLE `organization_role_user`
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织角色用户表';

CREATE TABLE `menu`
(
    `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
    `app_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id',
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
    KEY              `idx_app_id` (`app_id`),
    KEY              `idx_app_parent_id` (`app_id`, `parent_id`),
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
    `id`                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '连接器ID',
    `tenant_id`                BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`                     VARCHAR(128) NOT NULL DEFAULT '' COMMENT '连接器名称',
    `display_name`             VARCHAR(128) NOT NULL DEFAULT '' COMMENT '显示名称',
    `protocol`                 VARCHAR(64) NOT NULL DEFAULT '' COMMENT '协议类型',
    `provider`                 VARCHAR(128) NOT NULL DEFAULT '' COMMENT '提供商',
    `status`                   VARCHAR(32) NOT NULL DEFAULT '' COMMENT '状态',
    `allow_auto_create_user`   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许自动创建用户',
    `allow_account_link`       TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许账号关联',
    `sync_profile`             TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否同步资料',
    `enable_token_storage`     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用令牌存储',
    `config`                   JSON NOT NULL DEFAULT ('{}') COMMENT '连接器配置',
    `claim_mapping`            JSON NOT NULL DEFAULT ('{}') COMMENT '声明映射',
    `domain_policy`            JSON NOT NULL DEFAULT ('{}') COMMENT '域策略',
    `created_at`               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`               DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY                        `idx_tenant_id` (`tenant_id`),
    KEY                        `idx_tenant_name` (`tenant_id`, `name`),
    KEY                        `idx_tenant_provider` (`tenant_id`, `provider`),
    KEY                        `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='连接器表';

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
    `person_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '自然人ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `user_id`         BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户ID',
    `application_client_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'OIDC客户端ID',
    `session_id`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '会话ID',
    `token`           VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'token哈希',
    `client_type`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '客户端类型',
    `client_ip`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
    `user_agent`      VARCHAR(512) NOT NULL DEFAULT '' COMMENT '用户代理信息',
    `expired_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `revoked_at`      DATETIME DEFAULT NULL COMMENT '撤销时间',
    `last_rotated_at` DATETIME DEFAULT NULL COMMENT '最后轮换时间',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY               `idx_person_id` (`person_id`),
    KEY               `idx_tenant_id` (`tenant_id`),
    KEY               `idx_user_id` (`user_id`),
    UNIQUE KEY        `uk_token` (`token`),
    KEY               `idx_session_id` (`session_id`),
    KEY               `idx_person_tenant_user_session` (`person_id`, `tenant_id`, `user_id`, `session_id`),
    KEY               `idx_expired_at` (`expired_at`),
    KEY               `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='刷新令牌表';

CREATE TABLE `api_key`
(
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `name`            VARCHAR(128) NOT NULL DEFAULT '' COMMENT '密钥名称',
    `key_hash`        VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥哈希(SHA256)',
    `key_prefix`      VARCHAR(16) NOT NULL DEFAULT '' COMMENT '密钥前缀(前7位)',
    `scope`           JSON NOT NULL DEFAULT ('{}') COMMENT '权限范围',
    `expired_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `last_used_at`    DATETIME DEFAULT NULL COMMENT '最后使用时间',
    `revoked_at`      DATETIME DEFAULT NULL COMMENT '撤销时间',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`      DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人ID',
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    KEY               `idx_tenant_id` (`tenant_id`),
    KEY               `idx_key_hash` (`key_hash`),
    KEY               `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API密钥表';

CREATE TABLE `domain`
(
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `domain`        VARCHAR(256) NOT NULL DEFAULT '' COMMENT '域名',
    `is_verified`   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否验证',
    `verified_at`   DATETIME DEFAULT NULL COMMENT '验证时间',
    `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`    DATETIME DEFAULT NULL COMMENT '删除时间',
    `created_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人id',
    `updated_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新人id',
    `deleted_by`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '删除人id',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_domain` (`tenant_id`, `domain`),
    KEY             `idx_domain` (`domain`),
    KEY             `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='域名表';

CREATE TABLE `application_client` (
    `id`                            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '客户端ID',
    `tenant_id`                     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `app_id`                BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id',
    `client_id`                     VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'OIDC客户端ID',
    `name`                          VARCHAR(256) NOT NULL DEFAULT '' COMMENT '客户端名称',
    `redirect_uris`                 JSON NOT NULL DEFAULT ('[]') COMMENT '授权回调地址',
    `post_logout_redirect_uris`     JSON NOT NULL DEFAULT ('[]') COMMENT '登出回调地址',
    `back_channel_logout_uri`       VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'OIDC背信道登出通知地址',
    `grant_types`                   JSON NOT NULL DEFAULT ('["authorization_code"]') COMMENT '授权类型',
    `response_types`                JSON NOT NULL DEFAULT ('["code"]') COMMENT '响应类型',
    `token_endpoint_auth_method`    VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic' COMMENT '令牌端点认证方式',
    `allowed_origins`               JSON NOT NULL DEFAULT ('[]') COMMENT 'CORS白名单',
    `require_pkce`                  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否强制PKCE',
    `require_auth_time`             TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要auth_time声明',
    `default_scopes`                JSON NOT NULL DEFAULT ('["openid","profile"]') COMMENT '默认权限范围',
    `access_token_ttl`              BIGINT NOT NULL DEFAULT 900 COMMENT '访问令牌有效期(秒)',
    `refresh_token_ttl`             BIGINT NOT NULL DEFAULT 2592000 COMMENT '刷新令牌有效期(秒)',
    `type`                          VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '客户端类型',
    `is_third_party`                TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否第三方应用',
    `status`                        VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `is_system`                     TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否系统内置',
    `created_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`                    DATETIME DEFAULT NULL,
    `created_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_app_id` (`app_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端表';

CREATE TABLE `application_client_secret` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `application_client_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端ID',
    `name`            VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥名称',
    `value_hash`      VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥哈希',
    `value_prefix`    VARCHAR(16) NOT NULL DEFAULT '' COMMENT '密钥前缀',
    `expired_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `revoked_at`      DATETIME DEFAULT NULL COMMENT '撤销时间',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    KEY `idx_application_client_id` (`application_client_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端密钥表';

CREATE TABLE `tenant_application` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `app_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用id',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `config`          JSON NOT NULL DEFAULT ('{}') COMMENT '租户级应用配置',
    `granted_scope`   JSON NOT NULL DEFAULT ('[]') COMMENT '租户级scope授权',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_app` (`tenant_id`, `app_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_app_id` (`app_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户应用订阅表';

CREATE TABLE `audit_log` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `actor_person_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作人person id',
    `actor_user_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作人user id',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `client_id`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端id',
    `action`          VARCHAR(64) NOT NULL DEFAULT '' COMMENT '动作标识',
    `target_type`     VARCHAR(64) NOT NULL DEFAULT '' COMMENT '目标类型',
    `target_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '目标id',
    `result`          VARCHAR(16) NOT NULL DEFAULT '' COMMENT '结果 success/failure',
    `ip`              VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'IP',
    `user_agent`      VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'UA',
    `detail`          TEXT COMMENT '详情',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    KEY `idx_actor_person` (`actor_person_id`),
    KEY `idx_tenant_action` (`tenant_id`, `action`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志表';

CREATE TABLE `session` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `person_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '自然人id',
    `session_id`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '会话id',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `client_ip`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'IP',
    `user_agent`      VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'UA',
    `login_time`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
    `last_active_at`  DATETIME DEFAULT NULL COMMENT '最后活跃',
    `revoked_at`      DATETIME DEFAULT NULL COMMENT '撤销时间',
    `status`          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/revoked',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_session_id` (`session_id`),
    KEY `idx_person_id` (`person_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='会话审计表';
