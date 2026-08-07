# IAM 应用拆分设计方案

## 背景

当前 `backend/apps/iam` 应用暴露了约 150 个路由，混在一起的有：
- 认证相关路由（登录、注册、Token、OIDC Provider）
- 管理后台路由（用户/角色/菜单/租户/部门/应用等 CRUD）
- 租户自助管理路由（组织角色/组织用户 CRUD）

职责混杂，不利于独立部署、独立扩缩容和权限隔离。

## 拆分目标

将 `backend/apps/iam` 拆分为三个独立应用：

| 应用 | 命名 | 职责 |
|------|------|------|
| 认证网关 | `auth` | 登录/注册/Token/OIDC Provider/个人中心/会话管理 |
| IAM 平台管理 | `platformadmin` | 用户/角色/菜单/租户/部门/应用/API Key/域名/连接器/审计日志等超级管理 |
| 租户自助管理 | `tenantadmin` | 组织内角色管理/组织用户管理/组织角色-用户关联管理 |

## 数据库

三个应用共享同一套 IAM 数据库（同一组表）。

## 代码共享

将 `model`、`dao`、`object` 三层提取到 `backend/pkg/iam/` 作为共享包，三个应用各自 import 使用。

## 目录结构

```
backend/
├── apps/
│   ├── auth/                    # 认证网关
│   │   ├── cmd/main.go
│   │   ├── config/
│   │   ├── internal/
│   │   │   ├── controller/      # ctrauth, ctrperson, ctroidc, ctrsession
│   │   │   ├── service/         # svcauth, svcperson, svcoidc, svcsession
│   │   │   ├── dto/             # dtoauth, dtoperson, dtooidc, dtouser(session)
│   │   │   ├── router/          # auth/oidc/person/session 路由
│   │   │   └── middleware/      # oidcauth, silent_oidc, apikey_auth
│   │   ├── docs/
│   │   ├── go.mod
│   │   └── testutil/
│   ├── platformadmin/           # IAM 平台管理后台
│   │   ├── cmd/main.go
│   │   ├── config/
│   │   ├── internal/
│   │   │   ├── controller/      # ctruser, ctrtenant, ctrpermission, ctrapplication, ctrapikey, ctroauthclient, ctrdomain, ctrtenantapplication
│   │   │   ├── service/         # svcuser, svctenant, svcpermission, svcapplication, svcapikey, svcoauthclient, svcdomain
│   │   │   ├── dto/             # dtouser, dtotenant, dtopermission, dtoapplication, dtoapikey, dtooauthclient, dtodomain, dtotenantapplication, dtoconnector
│   │   │   ├── router/          # user/tenant/permission/application/api_key/oauth_client/domain/connector/tenant_application/log 路由
│   │   │   └── middleware/      # oidcauth
│   │   ├── docs/
│   │   ├── go.mod
│   │   └── testutil/
│   └── tenantadmin/             # 租户自助管理
│       ├── cmd/main.go
│       ├── config/
│       ├── internal/
│       │   ├── controller/      # ctrtenant（organizationRole/organizationUser/organizationRoleUser）
│       │   ├── service/         # svctenant（organizationRole/organizationUser/organizationRoleUser）
│       │   ├── dto/             # dtotenant（organization 相关）
│       │   ├── router/          # organizationRole/organizationUser/organizationRoleUser 路由
│       │   └── middleware/      # oidcauth
│       ├── docs/
│       ├── go.mod
│       └── testutil/
├── pkg/
│   ├── iam/                     # IAM 共享包
│   │   ├── model/               # 所有 IAM model（从 apps/iam/model 迁移）
│   │   ├── dao/                 # 所有 IAM dao（从 apps/iam/dao 迁移）
│   │   └── object/              # 所有 IAM object（从 apps/iam/object 迁移）
```

## 路由重新分配

### auth 应用 — 不加模块前缀，直接 `/v1/auth/{operation}`

| 旧路径 | 新路径 | 说明 |
|--------|--------|------|
| `/v1/auth/login` | `/v1/auth/login` | 登录 |
| `/v1/auth/register` | `/v1/auth/register` | 注册 |
| `/v1/auth/myTenants` | `/v1/auth/myTenants` | 我的租户 |
| `/v1/auth/selectTenant` | `/v1/auth/selectTenant` | 选择租户 |
| `/v1/auth/switchTenant` | `/v1/auth/switchTenant` | 切换租户 |
| `/v1/auth/joinTenant` | `/v1/auth/joinTenant` | 加入租户 |
| `/v1/auth/refreshToken` | `/v1/auth/refreshToken` | 刷新 Token |
| `/v1/auth/logout` | `/v1/auth/logout` | 登出 |
| `/v1/auth/logoutAll` | `/v1/auth/logoutAll` | 全部登出 |
| `/v1/auth/userinfo` | `/v1/auth/userinfo` | 当前用户信息 |
| `/v1/connector/callback` | `/v1/auth/connector/callback` | 连接器回调 |
| `/v1/oidc/*` | `/v1/auth/oidc/*` | OIDC Provider 全部端点 |
| `/v1/person/detail` | `/v1/auth/person/detail` | 个人资料 |
| `/v1/person/updatePassword` | `/v1/auth/person/updatePassword` | 修改个人密码 |
| `/v1/user/sessions` | `/v1/auth/user/sessions` | 会话管理 |

### platformadmin 应用 — `/v1/platformadmin/{module}/{operation}`

| 旧路径 | 新路径 |
|--------|--------|
| `/v1/tenant/*` | `/v1/platformadmin/tenant/*` |
| `/v1/department/*` | `/v1/platformadmin/department/*` |
| `/v1/organization/*` | `/v1/platformadmin/organization/*` |
| `/v1/system/*` | `/v1/platformadmin/system/*` |
| `/v1/user/*` | `/v1/platformadmin/user/*` |
| `/v1/role/*` | `/v1/platformadmin/role/*` |
| `/v1/menu/*` | `/v1/platformadmin/menu/*` |
| `/v1/scope/*` | `/v1/platformadmin/scope/*` |
| `/v1/resource/*` | `/v1/platformadmin/resource/*` |
| `/v1/roleMenu/*` | `/v1/platformadmin/roleMenu/*` |
| `/v1/roleScope/*` | `/v1/platformadmin/roleScope/*` |
| `/v1/userRole/*` | `/v1/platformadmin/userRole/*` |
| `/v1/oauthClient/*` | `/v1/platformadmin/oauthClient/*` |
| `/v1/application/*` | `/v1/platformadmin/application/*` |
| `/v1/apiKey/*` | `/v1/platformadmin/apiKey/*` |
| `/v1/domain/*` | `/v1/platformadmin/domain/*` |
| `/v1/connector/*`（除 callback） | `/v1/platformadmin/connector/*` |
| `/v1/tenantApplication/*` | `/v1/platformadmin/tenantApplication/*` |
| `/v1/log/*` | `/v1/platformadmin/log/*` |

### tenantadmin 应用 — `/v1/tenantadmin/{module}/{operation}`

| 旧路径 | 新路径 |
|--------|--------|
| `/v1/organizationRole/*` | `/v1/tenantadmin/organizationRole/*` |
| `/v1/organizationUser/*` | `/v1/tenantadmin/organizationUser/*` |
| `/v1/organizationRoleUser/*` | `/v1/tenantadmin/organizationRoleUser/*` |

## 中间件策略

| 应用 | 中间件 |
|------|--------|
| auth | OIDC 兼容认证（全局）+ Token 黑名单检查（全局），登录/注册/OIDC/connector 回调路径跳过认证 |
| platformadmin | OIDC 兼容认证（全局）+ 管理员权限校验，所有路由需要认证 |
| tenantadmin | OIDC 兼容认证（全局）+ 租户管理员权限校验，所有路由需要认证 |

## 路由路径变更汇总

- auth 应用：旧路由前添加 `/v1/auth/`（如 `/v1/person/detail` → `/v1/auth/person/detail`），原 `/v1/auth/*` 路由路径不变
- platformadmin 应用：旧路由前添加 `/v1/platformadmin/`
- tenantadmin 应用：旧路由前添加 `/v1/tenantadmin/`

## 构建命令

```bash
make build APP=auth
make build APP=platformadmin
make build APP=tenantadmin
make run APP=auth
make run APP=platformadmin
make run APP=tenantadmin
make test APP=auth
make test APP=platformadmin
make test APP=tenantadmin
```
