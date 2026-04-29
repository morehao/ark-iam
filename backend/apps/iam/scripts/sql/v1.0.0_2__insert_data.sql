-- ============================================
-- 平台管理员初始化脚本
-- 执行顺序：1.组织 2.租户 3.部门 4.自然人 5.用户
-- 密码格式：pwd + 邮箱（实际密码哈希需在应用层生成）
-- ============================================

USE ark_iam;

-- 定义常量
SET @platform_org_code = 'PLATFORM';
SET @platform_display_code = 'PLATFORM';
SET @platform_tenant_code = 'PLATFORM';
SET @platform_dept_code = 'PLATFORM';
SET @platform_admin_username = 'platform_admin';
SET @platform_admin_mobile = '13800138000';
SET @platform_admin_email = 'admin@platform.com';
SET @platform_admin_real_name = '平台管理员';

-- 检查是否已存在平台组织
SET @org_exists = (SELECT COUNT(*) FROM iam_organization WHERE code = @platform_org_code);

-- 1. 创建平台组织
INSERT INTO iam_organization
(code, display_code, org_name, description, status, created_at, updated_at)
SELECT @platform_org_code, @platform_display_code, '平台管理组织', '平台系统管理专用组织', 'enabled', NOW(), NOW()
FROM DUAL WHERE @org_exists = 0;

-- 获取平台组织ID
SET @org_id = (SELECT id FROM iam_organization WHERE code = @platform_org_code LIMIT 1);

-- 检查是否已存在平台租户
SET @tenant_exists = (SELECT COUNT(*) FROM iam_tenant WHERE tenant_code = @platform_tenant_code);

-- 2. 创建平台租户
INSERT INTO iam_tenant
(org_id, tenant_code, tenant_name, domain, status, created_at, updated_at)
SELECT @org_id, @platform_tenant_code, '平台管理租户', 'platform', 'enabled', NOW(), NOW()
FROM DUAL WHERE @tenant_exists = 0;

-- 获取平台租户ID
SET @tenant_id = (SELECT id FROM iam_tenant WHERE tenant_code = @platform_tenant_code LIMIT 1);

-- 检查是否已存在平台部门
SET @dept_exists = (SELECT COUNT(*) FROM iam_department WHERE tenant_id = @tenant_id AND dept_code = @platform_dept_code);

-- 3. 创建平台部门（与租户同名）
INSERT INTO iam_department
(tenant_id, dept_code, dept_name, dept_level, parent_id, status, created_at, updated_at)
SELECT @tenant_id, @platform_dept_code, '平台管理租户', 1, 0, 'enabled', NOW(), NOW()
FROM DUAL WHERE @dept_exists = 0;

-- 获取平台部门ID
SET @dept_id = (SELECT id FROM iam_department WHERE tenant_id = @tenant_id AND dept_code = @platform_dept_code LIMIT 1);

-- 更新部门路径（自引用路径）
UPDATE iam_department SET dept_path = CONCAT('/', @dept_id, '/') WHERE id = @dept_id AND (dept_path IS NULL OR dept_path = '');

-- 检查是否已存在管理员自然人
SET @person_exists = (SELECT COUNT(*) FROM iam_person WHERE email = @platform_admin_email);

-- 4. 创建管理员自然人
-- 注意：密码哈希需要在应用层生成，此处使用占位符
-- 实际密码应为：pwd + admin@platform.com = pwdadmin@platform.com
INSERT INTO iam_person 
(real_name, mobile, email, password_hash, created_at, updated_at)
SELECT @platform_admin_real_name, @platform_admin_mobile, @platform_admin_email, '$2a$10$PLACEHOLDER_REPLACE_WITH_APP_GENERATED_HASH', NOW(), NOW()
FROM DUAL WHERE @person_exists = 0;

-- 获取管理员自然人ID
SET @person_id = (SELECT id FROM iam_person WHERE email = @platform_admin_email LIMIT 1);

-- 检查是否已存在平台管理员账号
SET @user_exists = (SELECT COUNT(*) FROM iam_user WHERE username = @platform_admin_username);

-- 5. 创建平台管理员账号
INSERT INTO iam_user
(tenant_id, person_id, dept_id, username, user_type, status, created_at, updated_at)
SELECT @tenant_id, @person_id, @dept_id, @platform_admin_username, 'platform_admin', 'enabled', NOW(), NOW()
FROM DUAL WHERE @user_exists = 0;

-- 输出结果
SELECT 
    '初始化完成' AS status,
    @org_id AS org_id,
    @tenant_id AS tenant_id,
    @dept_id AS department_id,
    @person_id AS person_id,
    (SELECT id FROM iam_user WHERE username = @platform_admin_username LIMIT 1) AS user_id,
    @platform_admin_username AS username,
    @platform_admin_email AS email;