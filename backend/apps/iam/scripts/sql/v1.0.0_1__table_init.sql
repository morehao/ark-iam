-- ============================================================
-- IAM 数据库初始化脚本
-- 包含：创建数据库 + 建表语句 + 表结构迁移
-- ============================================================

-- 创建数据库
CREATE DATABASE IF NOT EXISTS ark_iam DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
USE ark_iam;

-- ============================================================
-- 第一部分：建表语句
-- ============================================================

-- 1. 应用表
-- ============================================
CREATE TABLE IF NOT EXISTS iam_application (
    id BIGINT AUTO_INCREMENT COMMENT '应用ID',
    app_code VARCHAR(32) UNIQUE NOT NULL COMMENT '应用编码',
    app_name VARCHAR(64) NOT NULL COMMENT '应用名称',
    app_type VARCHAR(16) DEFAULT 'web' COMMENT '应用类型: web-网页 app-移动端 mini-小程序',
    description VARCHAR(255) COMMENT '应用描述',
    homepage_url VARCHAR(255) COMMENT '应用首页URL',
    callback_url VARCHAR(255) COMMENT '回调URL',
    logo VARCHAR(255) COMMENT '应用Logo',
    sequence INT DEFAULT 0 COMMENT '排序',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (id),
    UNIQUE KEY uk_app_code (app_code),
    INDEX idx_status (status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用表';

-- 组织应用关联表
CREATE TABLE IF NOT EXISTS iam_organization_application (
    id BIGINT AUTO_INCREMENT COMMENT '关联ID',
    org_id BIGINT NOT NULL COMMENT '组织ID',
    app_id BIGINT NOT NULL COMMENT '应用ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    PRIMARY KEY (id),
    UNIQUE KEY uk_org_app (org_id, app_id),
    INDEX idx_org_id (org_id),
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织应用关联表';

-- 租户应用关联表
CREATE TABLE IF NOT EXISTS iam_tenant_application (
    id BIGINT AUTO_INCREMENT COMMENT '关联ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID',
    app_id BIGINT NOT NULL COMMENT '应用ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    PRIMARY KEY (id),
    UNIQUE KEY uk_tenant_app (tenant_id, app_id),
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户应用关联表';

-- 2. 租户核心表
-- ============================================

-- 组织表(最高层级)
CREATE TABLE IF NOT EXISTS iam_organization (
    id BIGINT AUTO_INCREMENT COMMENT '组织ID',
    code VARCHAR(32) NOT NULL COMMENT '系统内部唯一标识',
    display_code VARCHAR(32) UNIQUE NOT NULL COMMENT '组织编码(对外展示)',
    org_name VARCHAR(64) NOT NULL COMMENT '组织名称',
    description VARCHAR(255) COMMENT '组织描述',
    logo VARCHAR(255) COMMENT '组织Logo',
    sequence INT DEFAULT 0 COMMENT '排序',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_code (code),
    UNIQUE KEY uk_display_code (display_code),
    INDEX idx_created_at (created_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织表';

-- 组织配置表(统一配置表)
CREATE TABLE IF NOT EXISTS iam_organization_config (
    id BIGINT AUTO_INCREMENT COMMENT '配置ID',
    org_id BIGINT NOT NULL COMMENT '组织ID',
    config_key VARCHAR(100) NOT NULL COMMENT '配置键',
    config_value TEXT COMMENT '配置值(支持JSON)',
    value_type VARCHAR(32) DEFAULT 'string' COMMENT '配置类型: string/json/boolean/number',
    config_group VARCHAR(32) DEFAULT 'general' COMMENT '配置分组: general-通用/auth-认证/theme-主题等',
    description VARCHAR(255) NOT NULL DEFAULT '' COMMENT '配置说明',
    sequence INT DEFAULT 0 COMMENT '排序',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_org_config (org_id, config_group, config_key),
    INDEX idx_org_id (org_id),
    INDEX idx_config_group (config_group),
    INDEX idx_config_key (config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='组织配置表';

-- 租户表(租户主体)
CREATE TABLE IF NOT EXISTS iam_tenant (
    id BIGINT AUTO_INCREMENT COMMENT '租户ID',
    org_id BIGINT NOT NULL COMMENT '所属组织ID',
    parent_id BIGINT DEFAULT 0 COMMENT '父租户ID',
    tenant_code VARCHAR(32) NOT NULL COMMENT '租户编码',
    tenant_name VARCHAR(128) NOT NULL COMMENT '租户名称',
    tenant_level INT DEFAULT 1 COMMENT '租户层级',
    domain VARCHAR(255) DEFAULT '' COMMENT '租户域名(用于注册时子域名匹配)',
    tenant_path VARCHAR(512) COMMENT '租户路径: /1/2/3/',
    short_name VARCHAR(64) COMMENT '租户简称',
    legal_person VARCHAR(32) COMMENT '法人代表',
    contact_phone VARCHAR(16) COMMENT '联系电话',
    contact_email VARCHAR(64) COMMENT '联系邮箱',
    address VARCHAR(255) COMMENT '租户地址',
    logo VARCHAR(255) COMMENT '租户Logo',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 trial-试用 expired-已过期 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_org_tenant_code (org_id, tenant_code),
    UNIQUE KEY uk_org_domain (org_id, domain),
    INDEX idx_org_id (org_id),
    INDEX idx_org_tenant_status (org_id, status),
    INDEX idx_parent_id (parent_id),
    INDEX idx_tenant_path (tenant_path(100)),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户表';

-- 自然人表(跨租户的人员身份)
CREATE TABLE IF NOT EXISTS iam_person (
    id BIGINT AUTO_INCREMENT COMMENT '自然人ID',
    real_name VARCHAR(32) NOT NULL COMMENT '真实姓名',
    gender VARCHAR(8) COMMENT '性别: male-男 female-女 unknown-未知',
    birth_date DATE COMMENT '出生日期',
    mobile VARCHAR(16) COMMENT '手机号',
    email VARCHAR(64) COMMENT '邮箱',
    wechat VARCHAR(32) COMMENT '微信号',
    password_hash VARCHAR(128) COMMENT '密码哈希(不存储盐值,盐值在应用层生成)',
    avatar_url VARCHAR(255) COMMENT '头像URL',
    remark VARCHAR(500) COMMENT '备注',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    INDEX idx_real_name (real_name),
    INDEX idx_mobile (mobile),
    INDEX idx_email (email),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='自然人表';

-- 部门表
CREATE TABLE IF NOT EXISTS iam_department (
    id BIGINT AUTO_INCREMENT COMMENT '部门ID',
    tenant_id BIGINT NOT NULL COMMENT '所属租户ID',
    parent_id BIGINT DEFAULT 0 COMMENT '父部门ID,0表示根部门',
    dept_code VARCHAR(32) NOT NULL COMMENT '部门编码',
    dept_name VARCHAR(64) NOT NULL COMMENT '部门名称',
    dept_path VARCHAR(512) COMMENT '部门路径: /1/2/3/',
    dept_level INT DEFAULT 1 COMMENT '部门层级',
    leader_id BIGINT COMMENT '部门负责人ID',
    sequence INT DEFAULT 0 COMMENT '排序',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_tenant_code (tenant_id, dept_code),
    INDEX idx_tenant_parent (tenant_id, parent_id),
    INDEX idx_tenant_path (tenant_id, dept_path(100)),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_leader (leader_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门表';

-- 用户账号表(租户内的账号或平台管理员账号,一个自然人可在多个租户有账号)
CREATE TABLE IF NOT EXISTS iam_user (
    id BIGINT AUTO_INCREMENT COMMENT '用户ID',
    person_id BIGINT NOT NULL COMMENT '自然人ID',
    tenant_id BIGINT NOT NULL DEFAULT 0 COMMENT '所属租户ID',
    dept_id BIGINT COMMENT '主部门ID(冗余字段,方便查询,实际关联关系在iam_user_department表)',
    username VARCHAR(32) NOT NULL COMMENT '用户名(租户用户:租户内唯一,平台管理员:全局唯一,需应用层保证)',
    employee_no VARCHAR(32) COMMENT '工号',
    position VARCHAR(64) COMMENT '职位',
    job_level VARCHAR(32) COMMENT '职级',
    entry_date DATE COMMENT '入职日期',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-正常 locked-锁定 disabled-禁用',
    user_type VARCHAR(16) DEFAULT 'normal' COMMENT '用户类型: normal-普通用户 tenant_admin-租户管理员 platform_admin-平台管理员(可管理所有租户)',
    last_login_at DATETIME(3) NULL COMMENT '最后登录时间',
    last_login_ip VARCHAR(45) COMMENT '最后登录IP(支持IPv6)',
    login_count INT DEFAULT 0 COMMENT '登录次数',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_tenant_username (tenant_id, username),
    UNIQUE KEY uk_tenant_employee_no (tenant_id, employee_no),
    INDEX idx_person_id (person_id),
    INDEX idx_tenant_dept (tenant_id, dept_id),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_tenant_user_type (tenant_id, user_type),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户账号表';

-- 用户部门关联表(支持用户跨部门,每个用户只能有一个主部门,需应用层保证)
CREATE TABLE IF NOT EXISTS iam_user_department (
    id BIGINT AUTO_INCREMENT COMMENT '关联ID',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID(冗余)',
    dept_id BIGINT NOT NULL COMMENT '部门ID',
    dept_type VARCHAR(16) DEFAULT 'primary' COMMENT '部门类型: primary-主部门 secondary-其他部门',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_user_dept (user_id, dept_id),
    INDEX idx_user_id (user_id),
    INDEX idx_tenant_dept (tenant_id, dept_id),
    INDEX idx_user_type (user_id, dept_type),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户部门关联表';

-- 角色表
CREATE TABLE IF NOT EXISTS iam_role (
    id BIGINT AUTO_INCREMENT COMMENT '角色ID',
    tenant_id BIGINT NOT NULL COMMENT '所属租户ID',
    role_code VARCHAR(32) NOT NULL COMMENT '角色编码',
    role_name VARCHAR(64) NOT NULL COMMENT '角色名称',
    role_type VARCHAR(16) DEFAULT 'custom' COMMENT '角色类型: custom-自定义 system-系统内置',
    description VARCHAR(255) COMMENT '角色描述',
    data_scope VARCHAR(16) DEFAULT 'all' COMMENT '数据权限范围: all-全部 dept_and_sub-本部门及以下 dept-本部门 self-仅本人 custom-自定义',
    sequence INT DEFAULT 0 COMMENT '排序',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_tenant_code (tenant_id, role_code),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色表';

-- 用户角色关联表
CREATE TABLE IF NOT EXISTS iam_user_role (
    id BIGINT AUTO_INCREMENT COMMENT '关联ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID(冗余)',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    role_id BIGINT NOT NULL COMMENT '角色ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_tenant_user_role (tenant_id, user_id, role_id),
    INDEX idx_tenant_user (tenant_id, user_id),
    INDEX idx_tenant_role (tenant_id, role_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户角色关联表';

-- 菜单表
CREATE TABLE IF NOT EXISTS iam_menu (
    id BIGINT AUTO_INCREMENT COMMENT '菜单ID',
    tenant_id BIGINT NOT NULL COMMENT '所属租户ID',
    app_id BIGINT NOT NULL DEFAULT 0 COMMENT '所属应用ID',
    parent_id BIGINT DEFAULT 0 COMMENT '父菜单ID',
    menu_code VARCHAR(32) NOT NULL COMMENT '菜单编码',
    menu_name VARCHAR(64) NOT NULL COMMENT '菜单名称',
    menu_type VARCHAR(16) DEFAULT 'directory' COMMENT '菜单类型: directory-目录 menu-菜单 button-按钮',
    route_path VARCHAR(255) COMMENT '路由地址',
    component_path VARCHAR(255) COMMENT '组件路径',
    permission VARCHAR(64) COMMENT '权限标识: sys:user:add',
    icon VARCHAR(64) COMMENT '菜单图标',
    sequence INT DEFAULT 0 COMMENT '排序',
    visibility VARCHAR(16) DEFAULT 'visible' COMMENT '可见性: visible-可见 hidden-隐藏',
    access_policy INT DEFAULT 1 COMMENT '访问策略位掩码: 1-全部人可见 2-需授权 4-组织管理员 8-租户管理员',
    cache_type VARCHAR(16) DEFAULT 'disabled' COMMENT '缓存类型: enabled-启用 disabled-禁用',
    link_type VARCHAR(16) DEFAULT 'internal' COMMENT '链接类型: internal-内部链接 external-外部链接',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (id),
    UNIQUE KEY uk_tenant_app_code (tenant_id, app_id, menu_code),
    INDEX idx_tenant_parent (tenant_id, parent_id),
    INDEX idx_tenant_app (tenant_id, app_id),
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单表';

-- 角色菜单关联表
CREATE TABLE IF NOT EXISTS iam_role_menu (
    id BIGINT AUTO_INCREMENT COMMENT '关联ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID(冗余)',
    role_id BIGINT NOT NULL COMMENT '角色ID',
    menu_id BIGINT NOT NULL COMMENT '菜单ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY uk_tenant_role_menu (tenant_id, role_id, menu_id),
    INDEX idx_tenant_role (tenant_id, role_id),
    INDEX idx_tenant_menu (tenant_id, menu_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色菜单关联表';

-- 操作日志表
CREATE TABLE IF NOT EXISTS iam_operation_log (
    id BIGINT AUTO_INCREMENT COMMENT '日志ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID',
    user_id BIGINT COMMENT '操作人ID',
    username VARCHAR(32) COMMENT '操作人账号',
    module VARCHAR(32) COMMENT '操作模块',
    operation VARCHAR(16) COMMENT '操作类型: create/update/delete/query',
    method VARCHAR(16) COMMENT '请求方法: GET/POST/PUT/DELETE等',
    request_id VARCHAR(64) COMMENT '请求ID(用于追踪请求链路)',
    request_url VARCHAR(512) COMMENT '请求URL',
    request_params TEXT COMMENT '请求参数(JSON格式)',
    response_result TEXT COMMENT '返回结果(JSON格式)',
    ip_address VARCHAR(45) COMMENT 'IP地址(支持IPv6)',
    user_agent VARCHAR(512) COMMENT '用户代理',
    status VARCHAR(16) DEFAULT 'success' COMMENT '操作状态: success-成功 failed-失败',
    error_msg VARCHAR(1000) COMMENT '错误信息',
    execute_time INT COMMENT '执行时长(ms)',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    INDEX idx_tenant_user (tenant_id, user_id),
    INDEX idx_tenant_created (tenant_id, created_at),
    INDEX idx_tenant_module (tenant_id, module),
    INDEX idx_request_id (request_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='操作日志表';

-- 登录日志表
CREATE TABLE IF NOT EXISTS iam_login_log (
    id BIGINT AUTO_INCREMENT COMMENT '日志ID',
    tenant_id BIGINT NOT NULL COMMENT '租户ID',
    user_id BIGINT COMMENT '用户ID',
    username VARCHAR(32) COMMENT '用户名',
    login_type VARCHAR(16) COMMENT '登录类型: password/sms/wechat',
    login_status VARCHAR(16) COMMENT '登录状态: success-成功 failed-失败',
    login_message VARCHAR(128) COMMENT '登录消息',
    ip_address VARCHAR(45) COMMENT 'IP地址(支持IPv6)',
    location VARCHAR(128) COMMENT '登录地点',
    browser VARCHAR(64) COMMENT '浏览器',
    os VARCHAR(64) COMMENT '操作系统',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间, NULL表示未删除',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (`id`),
    INDEX idx_tenant_user (tenant_id, user_id),
    INDEX idx_tenant_created (tenant_id, created_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='登录日志表';

-- ============================================
-- 第二部分：OIDC & SSO 相关表
-- ============================================

-- API Key 表
CREATE TABLE IF NOT EXISTS iam_api_key (
    id BIGINT AUTO_INCREMENT COMMENT 'ID',
    tenant_id BIGINT NOT NULL DEFAULT 0 COMMENT '租户ID',
    user_id BIGINT NOT NULL DEFAULT 0 COMMENT '关联用户ID',
    app_id BIGINT NOT NULL DEFAULT 0 COMMENT '应用ID',
    key_name VARCHAR(64) NOT NULL COMMENT '密钥名称',
    key_prefix VARCHAR(16) NOT NULL COMMENT '密钥前缀(ark_开头)',
    api_key TEXT NOT NULL COMMENT 'AES加密的API Key',
    access_policy VARCHAR(16) DEFAULT 'all' COMMENT '访问策略: all-所有 ip-IP限制',
    allowed_ips TEXT COMMENT '允许的IP列表(JSON)',
    scopes VARCHAR(255) COMMENT '权限范围',
    status VARCHAR(16) DEFAULT 'enabled' COMMENT '状态: enabled-启用 disabled-停用',
    last_used_at DATETIME(3) COMMENT '最后使用时间',
    expires_at DATETIME(3) COMMENT '过期时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间',
    created_by BIGINT NOT NULL DEFAULT 0 COMMENT '创建人ID',
    updated_by BIGINT NOT NULL DEFAULT 0 COMMENT '更新人ID',
    deleted_by BIGINT NOT NULL DEFAULT 0 COMMENT '删除人ID',
    PRIMARY KEY (id),
    INDEX idx_tenant_user (tenant_id, user_id),
    INDEX idx_app_id (app_id),
    INDEX idx_status (status),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API Key表';

-- OIDC 授权码表
CREATE TABLE IF NOT EXISTS iam_auth_code (
    id BIGINT AUTO_INCREMENT COMMENT 'ID',
    code VARCHAR(64) NOT NULL COMMENT '授权码',
    client_id VARCHAR(64) NOT NULL COMMENT 'Client ID',
    person_id BIGINT NOT NULL DEFAULT 0 COMMENT '自然人ID',
    tenant_id BIGINT NOT NULL DEFAULT 0 COMMENT '租户ID',
    org_id BIGINT NOT NULL DEFAULT 0 COMMENT '组织ID',
    redirect_uri VARCHAR(255) NOT NULL COMMENT '重定向URI',
    scope VARCHAR(255) DEFAULT 'openid,profile' COMMENT '请求的scope',
    state VARCHAR(128) COMMENT 'state参数，防CSRF',
    code_challenge VARCHAR(64) COMMENT 'PKCE code_challenge',
    code_challenge_method VARCHAR(8) DEFAULT 'S256' COMMENT 'PKCE challenge方法',
    expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
    used TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已使用',
    used_at DATETIME(3) COMMENT '使用时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_code (code),
    INDEX idx_client_id (client_id),
    INDEX idx_person_id (person_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC授权码表';

-- SSO 会话表
CREATE TABLE IF NOT EXISTS iam_sso_session (
    id BIGINT AUTO_INCREMENT COMMENT 'ID',
    session_id VARCHAR(64) NOT NULL COMMENT 'SSO会话ID',
    person_id BIGINT NOT NULL DEFAULT 0 COMMENT '自然人ID',
    org_id BIGINT NOT NULL DEFAULT 0 COMMENT '组织ID',
    login_time DATETIME(3) NOT NULL COMMENT '登录时间',
    last_active_time DATETIME(3) NOT NULL COMMENT '最后活跃时间',
    expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_session_id (session_id),
    INDEX idx_person_id (person_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='SSO会话表';

-- Token 表
CREATE TABLE IF NOT EXISTS iam_token (
    id BIGINT AUTO_INCREMENT COMMENT 'ID',
    token_id VARCHAR(64) NOT NULL COMMENT 'Token唯一标识',
    person_id BIGINT NOT NULL DEFAULT 0 COMMENT '自然人ID',
    user_id BIGINT NOT NULL DEFAULT 0 COMMENT '用户ID',
    client_id VARCHAR(64) NOT NULL COMMENT 'Client ID',
    tenant_id BIGINT NOT NULL DEFAULT 0 COMMENT '租户ID',
    org_id BIGINT NOT NULL DEFAULT 0 COMMENT '组织ID',
    token_type VARCHAR(16) NOT NULL COMMENT 'Token类型: access/refresh/id',
    access_token_hash VARCHAR(128) COMMENT 'Access Token哈希',
    refresh_token_hash VARCHAR(128) COMMENT 'Refresh Token哈希',
    scopes VARCHAR(255) DEFAULT 'openid,profile' COMMENT '授权的scopes',
    expires_at DATETIME(3) NOT NULL COMMENT '过期时间',
    revoked TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否撤销',
    revoked_at DATETIME(3) COMMENT '撤销时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_token_id (token_id),
    INDEX idx_person_id (person_id),
    INDEX idx_user_id (user_id),
    INDEX idx_client_id (client_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_revoked (revoked),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Token表';
