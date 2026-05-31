# Application 模型拆分设计文档

- **日期**: 2026-06-01
- **状态**: 草案
- **前置文档**: `2026-05-31-application-oidc-design.md`（OIDC Provider 设计）

## 1. 背景与问题

### 1.1 场景描述

- 平台有多个应用（如 CMS、CRM），各应用有独立的菜单和权限体系
- 一个租户可能使用多个应用
- 每个应用在不同租户中的菜单结构是一致的
- 角色可以按（租户, 应用）维度独立定义

### 1.2 当前设计的问题

当前 `application` 表将两个概念混在一起：

1. **OIDC 客户端注册** — client_id、redirect_uris、grant_types 等协议配置
2. **应用定义** — name、description、logo、homepage

同时 `menu` 表作用域为 `tenant_id`（每个租户各自维护菜单），导致：

- 同一应用的菜单在不同租户间重复维护
- 菜单修改需要同步到所有租户
- 角色没有应用维度，无法按（租户, 应用）隔离权限

### 1.3 目标

将"应用定义"与"OIDC 客户端"两个概念分离，引入租户应用订阅机制，使菜单和角色具备应用维度隔离。

### 1.4 非目标

- 不改变 OIDC 认证流程本身（由前置文档 `2026-05-31-application-oidc-design.md` 定义）
- 不改变现有 user、person、department 等核心模型
- 不涉及前端 UI 变化

## 2. 核心概念

### 2.1 三表结构

| 表 | 作用域 | 谁维护 | 职责 |
|---|---|---|---|
| `application`（应用定义） | 全局 | 平台管理员 | 定义平台上有哪些应用可用（CMS、CRM） |
| `tenant_application`（租户应用订阅） | 租户级 | 租户管理员 | 记录租户订阅了哪些应用及配置 |
| `oauth_client`（OIDC 客户端） | 租户级 | 租户管理员 | OIDC 协议级别客户端注册（client_id、密钥、回调） |

### 2.2 概念关系

```
application（全局应用定义）
  ├── code: "cms"
  ├── name: "CMS 内容管理系统"
  ├── menus: [菜单树]（全局定义，所有租户共享）
  │
  ├── tenant_application（租户 A 订阅）
  │   ├── status: enable
  │   ├── config: { custom_logo, domain }
  │   │
  │   └── oauth_client（可选）
  │       ├── client_id: xxx
  │       ├── redirect_uris: [...]
  │       └── secrets: [...]
  │
  └── tenant_application（租户 B 订阅）
      └── ...
```

## 3. 表结构设计

### 3.1 application（全局应用定义）

```sql
CREATE TABLE `application` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '应用ID',
    `code`            VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用编码（唯一）',
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
```

### 3.2 tenant_application（租户应用订阅）

```sql
CREATE TABLE `tenant_application` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用id',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `config`          JSON NOT NULL DEFAULT ('{}') COMMENT '租户级应用配置',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_app` (`tenant_id`, `application_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_application_id` (`application_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户应用订阅表';
```

### 3.3 oauth_client（OIDC 客户端）

从当前 `application` 表迁移 OIDC 字段得到：

```sql
CREATE TABLE `oauth_client` (
    `id`                            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '客户端ID',
    `tenant_id`                     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`                BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id',
    `client_id`                     VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'OIDC客户端ID',
    `name`                          VARCHAR(256) NOT NULL DEFAULT '' COMMENT '客户端名称',
    `redirect_uris`                 JSON NOT NULL DEFAULT ('[]') COMMENT '授权回调地址',
    `post_logout_redirect_uris`     JSON NOT NULL DEFAULT ('[]') COMMENT '登出回调地址',
    `grant_types`                   JSON NOT NULL DEFAULT ('["authorization_code"]') COMMENT '授权类型',
    `response_types`                JSON NOT NULL DEFAULT ('["code"]') COMMENT '响应类型',
    `token_endpoint_auth_method`    VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic' COMMENT '令牌端点认证方式',
    `allowed_origins`               JSON NOT NULL DEFAULT ('[]') COMMENT 'CORS白名单',
    `require_pkce`                  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否强制PKCE',
    `require_auth_time`             TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要auth_time声明',
    `default_scopes`                JSON NOT NULL DEFAULT ('["openid","profile"]') COMMENT '默认权限范围',
    `access_token_ttl`              BIGINT NOT NULL DEFAULT 3600 COMMENT '访问令牌有效期(秒)',
    `refresh_token_ttl`             BIGINT NOT NULL DEFAULT 2592000 COMMENT '刷新令牌有效期(秒)',
    `type`                          VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '客户端类型',
    `is_third_party`                TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否第三方应用',
    `status`                        VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `created_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`                    DATETIME DEFAULT NULL,
    `created_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_application_id` (`application_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端表';
```

### 3.4 oauth_client_secret（客户端密钥）

```sql
CREATE TABLE `oauth_client_secret` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `oauth_client_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端ID',
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
    KEY `idx_oauth_client_id` (`oauth_client_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端密钥表';
```

## 4. 菜单模型调整

### 4.1 menu 表变更

| 字段 | 当前 | 新设计 |
|------|------|--------|
| `tenant_id` | 有 | 移除 |
| `application_id` | 无 | 新增（关联全局应用） |

`menu` 表从租户作用域改为应用作用域。应用级菜单由平台管理员维护，所有订阅该应用的租户共享同一套菜单结构。租户通过 `role_menu` 控制角色可访问的菜单子集。

### 4.2 菜单数据流

```
平台管理员创建 CMS 应用
  └── 定义 CMS 菜单树（应用级）
      ├── 内容管理
      │   ├── 文章列表
      │   └── 分类管理
      ├── 用户管理
      └── 系统设置

租户 A 订阅 CMS
  └── 获取 CMS 菜单树
  └── 创建角色 "CMS编辑" → 分配菜单子集（内容管理）
  └── 创建角色 "CMS管理员" → 分配全部菜单

租户 B 订阅 CMS
  └── 获取相同的 CMS 菜单树
  └── 创建角色 "CMS查看者" → 仅分配部分菜单
```

## 5. 角色模型调整

### 5.1 role 表变更

| 字段 | 当前 | 新设计 |
|------|------|--------|
| `application_id` | 无 | 新增（可空，关联全局应用） |

`application_id` 为空表示租户级角色（跨应用），不为空表示该应用下的角色：

```
role 数据示例:
┌──────────┬────────────────┬────────────────────────┐
│ tenant_id│ application_id │ role                    │
├──────────┼────────────────┼────────────────────────┤
│ 租户A    │ CMS            │ CMS 管理员, CMS 编辑    │
│ 租户A    │ CRM            │ CRM 管理员, CRM 销售    │
│ 租户B    │ CMS            │ CMS 查看者              │
│ 租户A    │ NULL           │ 租户超级管理员           │
└──────────┴────────────────┴────────────────────────┘
```

### 5.2 受影响关联表

| 表 | 变更 |
|---|---|
| `application_role` | 移除（角色已有 `application_id`，天然按应用隔离） |
| `role_menu` | 不变（与菜单关联逻辑不变） |
| `user_role` | 不变（用户-角色关联不变） |

## 6. 数据迁移

### 6.1 存量数据迁移

当前 `application` 表的数据拆分为三条记录：

```
迁移前:
  application: id=1, tenant_id=1, name="CMS", client_id="xxx",
               redirect_uris=[...], grant_types=[...], ...

迁移后:
  application:          id=1, code="cms", name="CMS"（去重后入库）
  tenant_application:   tenant_id=1, application_id=1
  oauth_client:         tenant_id=1, application_id=1,
                        client_id="xxx", redirect_uris=[...], grant_types=[...]
```

迁移步骤：

1. 从当前 `application` 表中按 name 去重，生成 `application` 表数据
2. 遍历每条原记录，生成 `tenant_application` + `oauth_client`
3. `application_secret` 关联指向 `oauth_client.id`
4. `application_role` 数据迁移到 `role.application_id`
5. `menu` 表增加 `application_id` 字段，迁移关联关系
6. `refresh_token.application_id` 改为 `oauth_client_id`，关联 `oauth_client.id`

### 6.2 API 端点变更

| 当前路径 | 新路径 | 说明 |
|---|---|---|
| `POST /v1/application/create` | 保留（操作 oauth_client） | 当前 CRUD 逻辑保持不变 |
| `POST /v1/platform/application/create` | 新增 | 平台管理员创建全局应用定义 |
| `POST /v1/platform/application/menu/create` | 新增 | 平台管理员维护应用菜单 |

## 7. 影响范围

### 7.1 现有代码变动

| 模块 | 影响 |
|---|---|
| `model/application.go` | 拆分为 `application.go` + `oauth_client.go` |
| `model/application_secret.go` | 改为 `oauth_client_secret.go`，关联调整 |
| `model/application_role.go` | 移除 |
| `model/menu.go` | `tenant_id` → `application_id` |
| `model/role.go` | 新增 `application_id` |
| `dao/` | 对应调整 |
| `service/svcapplication/` | 拆分为 `svcapplication`（全局应用）+ `svcoauthclient`（OIDC 客户端） |
| `controller/` | 对应调整 |
| `dto/` | 对应调整 |
| OIDC 集成 (`svcoidc/`) | `GetClientByClientID` 改为查询 `oauth_client` 表 |

### 7.2 不受影响模块

- user、person、department、organization — 保持不变
- connector — 保持不变
- auth 认证流程 — 保持不变（仅 OIDC 端点查询目标表变化）
- refresh_token — `application_id` 字段名改为 `oauth_client_id`，关联目标变为 `oauth_client.id`

## 8. 实施步骤

### 第 1 步：新增 application（全局应用定义）+ tenant_application 表

- 创建新表 SQL
- 新增 model、dao、service、controller、dto
- 注册路由（平台管理端）

### 第 2 步：新建 oauth_client + oauth_client_secret 表

- 从当前 `application` 表拆分 OIDC 字段
- 新增 model、dao、service、controller、dto
- 改造现有 application 代码指向新表

### 第 3 步：数据迁移脚本

- 编写迁移脚本，执行存量数据拆分

### 第 4 步：menu 表 tenant_id → application_id

- 改表结构
- 更新 model、dao、业务逻辑

### 第 5 步：role 表新增 application_id + 移除 application_role

- 改表结构
- 更新 model、dao、业务逻辑
- 移除 `application_role` 相关代码

### 第 6 步：更新 OIDC 集成代码引用

- 更新 `svcoidc` 中对 application 表的查询
