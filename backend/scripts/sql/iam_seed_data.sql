-- IAM 种子数据初始化脚本
-- 执行顺序: 1) iam_schema.sql 2) iam_seed_data.sql

-- ============================================
-- 1. 租户种子数据
-- ============================================
INSERT INTO `tenant` (`id`, `name`, `code`, `type`, `db_user`, `is_suspended`, `tag`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 'Default Tenant', 'platform', 'platform', 'default_user', 0, 'default', 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `code` = VALUES(`code`), `type` = VALUES(`type`);

-- ============================================
-- 2. 基础角色种子数据
-- ============================================
INSERT INTO `role` (`id`, `tenant_id`, `app_id`, `name`, `code`, `description`, `type`, `is_default`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    (1, 1, 1, '管理员', 'admin', '系统管理员，拥有所有权限', 'User', 1, 0, 0, 0),
    (2, 1, 1, '普通用户', 'user', '普通用户，拥有基本查看权限', 'User', 1, 0, 0, 0),
    (3, 1, 1, '访客', 'guest', '访客，仅有只读权限', 'User', 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ============================================
-- 3. 基础资源种子数据
-- ============================================
INSERT INTO `resource` (`id`, `tenant_id`, `name`, `indicator`, `is_default`, `access_token_ttl`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    (1, 1, '管理后台', 'urn:ark:iam:admin', 1, 3600, 0, 0, 0),
    (2, 1, '用户中心', 'urn:ark:iam:me', 1, 3600, 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ============================================
-- 4. 基础权限(Scope)种子数据
-- ============================================
INSERT INTO `scope` (`id`, `tenant_id`, `resource_id`, `name`, `description`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    -- 管理后台权限
    (1, 1, 1, 'admin:user:read', '查看用户', 0, 0, 0),
    (2, 1, 1, 'admin:user:write', '管理用户', 0, 0, 0),
    (3, 1, 1, 'admin:role:read', '查看角色', 0, 0, 0),
    (4, 1, 1, 'admin:role:write', '管理角色', 0, 0, 0),
    (5, 1, 1, 'admin:menu:read', '查看菜单', 0, 0, 0),
    (6, 1, 1, 'admin:menu:write', '管理菜单', 0, 0, 0),
    (7, 1, 1, 'admin:department:read', '查看部门', 0, 0, 0),
    (8, 1, 1, 'admin:department:write', '管理部门', 0, 0, 0),
    (9, 1, 1, 'admin:application:read', '查看应用', 0, 0, 0),
    (10, 1, 1, 'admin:application:write', '管理应用', 0, 0, 0),
    (11, 1, 1, 'admin:resource:read', '查看资源', 0, 0, 0),
    (12, 1, 1, 'admin:resource:write', '管理资源', 0, 0, 0),
    -- 用户中心权限
    (13, 1, 2, 'me:profile:read', '查看个人信息', 0, 0, 0),
    (14, 1, 2, 'me:profile:write', '修改个人信息', 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ============================================
-- 5. 角色权限关联种子数据
-- ============================================
INSERT INTO `role_scope` (`id`, `tenant_id`, `role_id`, `scope_id`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    -- 管理员拥有所有权限
    (1, 1, 1, 1, 0, 0, 0),
    (2, 1, 1, 2, 0, 0, 0),
    (3, 1, 1, 3, 0, 0, 0),
    (4, 1, 1, 4, 0, 0, 0),
    (5, 1, 1, 5, 0, 0, 0),
    (6, 1, 1, 6, 0, 0, 0),
    (7, 1, 1, 7, 0, 0, 0),
    (8, 1, 1, 8, 0, 0, 0),
    (9, 1, 1, 9, 0, 0, 0),
    (10, 1, 1, 10, 0, 0, 0),
    (11, 1, 1, 11, 0, 0, 0),
    (12, 1, 1, 12, 0, 0, 0),
    (13, 1, 1, 13, 0, 0, 0),
    (14, 1, 1, 14, 0, 0, 0),
    -- 普通用户拥有用户中心权限
    (15, 1, 2, 13, 0, 0, 0),
    (16, 1, 2, 14, 0, 0, 0),
    -- 访客仅有查看权限
    (17, 1, 3, 1, 0, 0, 0),
    (18, 1, 3, 3, 0, 0, 0),
    (19, 1, 3, 5, 0, 0, 0),
    (20, 1, 3, 7, 0, 0, 0),
    (21, 1, 3, 9, 0, 0, 0),
    (22, 1, 3, 11, 0, 0, 0),
    (23, 1, 3, 13, 0, 0, 0)
ON DUPLICATE KEY UPDATE `role_id` = VALUES(`role_id`);

-- ============================================
-- 6. 基础菜单种子数据
-- ============================================
INSERT INTO `menu` (`id`, `app_id`, `parent_id`, `name`, `code`, `path`, `icon`, `sort`, `type`, `component`, `redirect`, `hidden`, `external_link`, `keep_alive`, `permission`, `status`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    -- 一级菜单
    (1, 1, 0, '工作台', 'dashboard', '/dashboard', 'dashboard', 1, 'menu', 'Layout', '', 0, 0, 0, '', 'enable', 0, 0, 0),
    (2, 1, 0, '用户管理', 'user', '/user', 'user', 2, 'menu', 'Layout', '', 0, 0, 0, 'admin:user:read', 'enable', 0, 0, 0),
    (3, 1, 0, '角色管理', 'role', '/role', 'role', 3, 'menu', 'Layout', '', 0, 0, 0, 'admin:role:read', 'enable', 0, 0, 0),
    (4, 1, 0, '菜单管理', 'menu', '/menu', 'menu', 4, 'menu', 'Layout', '', 0, 0, 0, 'admin:menu:read', 'enable', 0, 0, 0),
    (5, 1, 0, '部门管理', 'department', '/department', 'department', 5, 'menu', 'Layout', '', 0, 0, 0, 'admin:department:read', 'enable', 0, 0, 0),
    (6, 1, 0, '应用管理', 'application', '/application', 'app', 6, 'menu', 'Layout', '', 0, 0, 0, 'admin:application:read', 'enable', 0, 0, 0),
    (7, 1, 0, '资源管理', 'resource', '/resource', 'resource', 7, 'menu', 'Layout', '', 0, 0, 0, 'admin:resource:read', 'enable', 0, 0, 0),
    -- 用户管理子菜单
    (8, 1, 2, '用户列表', 'user-list', '/user/list', '', 1, 'menu', '/user/list/index', '', 0, 0, 0, 'admin:user:read', 'enable', 0, 0, 0),
    -- 角色管理子菜单
    (9, 1, 3, '角色列表', 'role-list', '/role/list', '', 1, 'menu', '/role/list/index', '', 0, 0, 0, 'admin:role:read', 'enable', 0, 0, 0),
    (10, 1, 3, '权限配置', 'role-permission', '/role/permission', '', 2, 'menu', '/role/permission/index', '', 0, 0, 0, 'admin:role:write', 'enable', 0, 0, 0),
    -- 菜单管理子菜单
    (11, 1, 4, '菜单列表', 'menu-list', '/menu/list', '', 1, 'menu', '/menu/list/index', '', 0, 0, 0, 'admin:menu:read', 'enable', 0, 0, 0),
    -- 部门管理子菜单
    (12, 1, 5, '部门列表', 'department-list', '/department/list', '', 1, 'menu', '/department/list/index', '', 0, 0, 0, 'admin:department:read', 'enable', 0, 0, 0),
    -- 应用管理子菜单
    (13, 1, 6, '应用列表', 'application-list', '/application/list', '', 1, 'menu', '/application/list/index', '', 0, 0, 0, 'admin:application:read', 'enable', 0, 0, 0),
    -- 资源管理子菜单
    (14, 1, 7, '资源列表', 'resource-list', '/resource/list', '', 1, 'menu', '/resource/list/index', '', 0, 0, 0, 'admin:resource:read', 'enable', 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ============================================
-- 7. 角色菜单关联种子数据
-- ============================================
INSERT INTO `role_menu` (`id`, `tenant_id`, `role_id`, `menu_id`, `created_by`, `updated_by`, `deleted_by`)
VALUES
    -- 管理员拥有所有菜单
    (1, 1, 1, 1, 0, 0, 0),
    (2, 1, 1, 2, 0, 0, 0),
    (3, 1, 1, 3, 0, 0, 0),
    (4, 1, 1, 4, 0, 0, 0),
    (5, 1, 1, 5, 0, 0, 0),
    (6, 1, 1, 6, 0, 0, 0),
    (7, 1, 1, 7, 0, 0, 0),
    (8, 1, 1, 8, 0, 0, 0),
    (9, 1, 1, 9, 0, 0, 0),
    (10, 1, 1, 10, 0, 0, 0),
    (11, 1, 1, 11, 0, 0, 0),
    (12, 1, 1, 12, 0, 0, 0),
    (13, 1, 1, 13, 0, 0, 0),
    (14, 1, 1, 14, 0, 0, 0),
    -- 普通用户拥有工作台和用户中心
    (15, 1, 2, 1, 0, 0, 0),
    (16, 1, 2, 2, 0, 0, 0),
    (17, 1, 2, 8, 0, 0, 0),
    -- 访客仅有工作台
    (18, 1, 3, 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `role_id` = VALUES(`role_id`);

-- ============================================
-- 8. 管理后台应用及角色关联种子数据
-- ============================================
INSERT INTO `application` (`id`, `code`, `name`, `description`, `type`, `status`, `sort`, `is_system`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 'admin', '管理后台', '平台管理后台应用', 'first_party', 'enable', 0, 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `is_system` = VALUES(`is_system`);

-- ============================================
-- 9. 租户应用订阅种子数据
-- ============================================
INSERT INTO `tenant_application` (`id`, `tenant_id`, `app_id`, `status`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 1, 1, 'enable', 0, 0, 0)
ON DUPLICATE KEY UPDATE `status` = VALUES(`status`);

-- ============================================
-- 10. 默认管理员用户种子数据
-- 密码: admin123 (bcrypt 加密)
-- ============================================
-- 10.1 先插入到 person 表
INSERT INTO `person` (`id`, `username`, `primary_email`, `primary_phone`, `password_encrypted`, `password_method`, `name`, `avatar`, `profile`, `custom_data`, `is_suspended`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 'admin', 'admin@example.com', '13800000000', '$2a$10$Js0KMHZVZY7z0kLoHAvckui8KJK5..xhKkzU2jwiz7X./aIANfnxi', 'bcrypt', '系统管理员', '', '{}', '{}', 0, 0, 0, 0)
ON DUPLICATE KEY UPDATE `username` = VALUES(`username`), `primary_email` = VALUES(`primary_email`), `primary_phone` = VALUES(`primary_phone`);

-- 10.2 再插入到 user 表（关联 person_id）
INSERT INTO `user` (`id`, `tenant_id`, `person_id`, `name`, `avatar`, `profile`, `custom_data`, `is_suspended`, `is_owner`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 1, 1, '系统管理员', '', '{}', '{}', 0, 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `tenant_id` = VALUES(`tenant_id`);

-- ============================================
-- 11. 管理员用户角色关联
-- ============================================
INSERT INTO `user_role` (`id`, `tenant_id`, `user_id`, `role_id`, `created_by`, `updated_by`, `deleted_by`)
VALUES (1, 1, 1, 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE `user_id` = VALUES(`user_id`);

-- ============================================
-- 12. OIDC SSO 测试客户端种子数据
-- ============================================

-- ============================================
-- platform-admin-web OAuth Client (第一方 SPA，PKCE)
-- client_id: platform-admin-web
-- redirect_uris: http://localhost:3001/auth/callback
-- back_channel_logout_uri: gateway 聚合部署下指向本应用接收端
-- ============================================
INSERT INTO `application_client` (`id`, `tenant_id`, `app_id`, `client_id`, `name`, `redirect_uris`, `grant_types`, `response_types`, `token_endpoint_auth_method`, `require_pkce`, `default_scopes`, `post_logout_redirect_uris`, `back_channel_logout_uri`, `type`, `is_third_party`, `status`, `is_system`, `created_by`, `updated_by`, `deleted_by`)
VALUES (2, 1, 1, 'platform-admin-web', 'IAM管理平台', '["http://localhost:3001/auth/callback"]', '["authorization_code","refresh_token"]', '["code"]', 'none', 1, '["openid","profile","email"]', '["http://localhost:3001/login"]', 'http://localhost:8100/oidc/bc-logout/platform', 'first_party', 0, 'enable', 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE
  `redirect_uris` = VALUES(`redirect_uris`),
  `grant_types` = VALUES(`grant_types`),
  `response_types` = VALUES(`response_types`),
  `token_endpoint_auth_method` = VALUES(`token_endpoint_auth_method`),
  `require_pkce` = VALUES(`require_pkce`),
  `default_scopes` = VALUES(`default_scopes`),
  `post_logout_redirect_uris` = VALUES(`post_logout_redirect_uris`),
  `back_channel_logout_uri` = VALUES(`back_channel_logout_uri`),
  `type` = VALUES(`type`),
  `is_third_party` = VALUES(`is_third_party`),
  `status` = VALUES(`status`),
  `is_system` = VALUES(`is_system`);

-- ============================================
-- tenant-admin-web OAuth Client (第一方 SPA，PKCE)
-- client_id: tenant-admin-web
-- redirect_uris: http://localhost:3002/auth/callback
-- back_channel_logout_uri: gateway 聚合部署下指向本应用接收端
-- ============================================
INSERT INTO `application_client` (`id`, `tenant_id`, `app_id`, `client_id`, `name`, `redirect_uris`, `grant_types`, `response_types`, `token_endpoint_auth_method`, `require_pkce`, `default_scopes`, `post_logout_redirect_uris`, `back_channel_logout_uri`, `type`, `is_third_party`, `status`, `is_system`, `created_by`, `updated_by`, `deleted_by`)
VALUES (3, 1, 1, 'tenant-admin-web', '租户管理平台', '["http://localhost:3002/auth/callback"]', '["authorization_code","refresh_token"]', '["code"]', 'none', 1, '["openid","profile","email"]', '["http://localhost:3002/login"]', 'http://localhost:8100/oidc/bc-logout/tenant', 'first_party', 0, 'enable', 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE
  `redirect_uris` = VALUES(`redirect_uris`),
  `grant_types` = VALUES(`grant_types`),
  `response_types` = VALUES(`response_types`),
  `token_endpoint_auth_method` = VALUES(`token_endpoint_auth_method`),
  `require_pkce` = VALUES(`require_pkce`),
  `default_scopes` = VALUES(`default_scopes`),
  `post_logout_redirect_uris` = VALUES(`post_logout_redirect_uris`),
  `back_channel_logout_uri` = VALUES(`back_channel_logout_uri`),
  `type` = VALUES(`type`),
  `is_third_party` = VALUES(`is_third_party`),
  `status` = VALUES(`status`),
  `is_system` = VALUES(`is_system`);

-- ============================================
-- login-web OAuth Client (第一方 SPA，PKCE，统一登录门户)
-- client_id: login-web
-- redirect_uris: http://localhost:3000/auth/callback
-- 定位：既作为 OP 跳转的集中登录页，也作为受 SSO 保护的门户 App（可触发全局登出）
-- ============================================
INSERT INTO `application_client` (`id`, `tenant_id`, `app_id`, `client_id`, `name`, `redirect_uris`, `grant_types`, `response_types`, `token_endpoint_auth_method`, `require_pkce`, `default_scopes`, `post_logout_redirect_uris`, `type`, `is_third_party`, `status`, `is_system`, `created_by`, `updated_by`, `deleted_by`)
VALUES (4, 1, 1, 'login-web', 'IAM 登录门户', '["http://localhost:3000/auth/callback"]', '["authorization_code","refresh_token"]', '["code"]', 'none', 1, '["openid","profile","email"]', '["http://localhost:3000/login"]', 'first_party', 0, 'enable', 1, 0, 0, 0)
ON DUPLICATE KEY UPDATE
  `redirect_uris` = VALUES(`redirect_uris`),
  `grant_types` = VALUES(`grant_types`),
  `response_types` = VALUES(`response_types`),
  `token_endpoint_auth_method` = VALUES(`token_endpoint_auth_method`),
  `require_pkce` = VALUES(`require_pkce`),
  `default_scopes` = VALUES(`default_scopes`),
  `post_logout_redirect_uris` = VALUES(`post_logout_redirect_uris`),
  `type` = VALUES(`type`),
  `is_third_party` = VALUES(`is_third_party`),
  `status` = VALUES(`status`),
  `is_system` = VALUES(`is_system`);
