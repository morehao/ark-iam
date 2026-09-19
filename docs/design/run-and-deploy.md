# 运行与部署指南（Run & Deploy）

> 本文说明 Ark IAM 的本地开发环境、构建运行、测试、Docker 部署与多环境部署要点。

---

## 目录

1. [环境要求](#1-环境要求)
2. [本地开发运行](#2-本地开发运行)
3. [测试](#3-测试)
4. [Docker 部署](#4-docker-部署)
5. [多环境部署拓扑](#5-多环境部署拓扑)
6. [常见排障](#6-常见排障)

---

## 1. 环境要求

| 依赖 | 版本 | 用途 |
|---|---|---|
| Go | 1.26+ | 后端（`backend/go.work` 与各 `go.mod` 均声明 1.26.1；Docker 构建镜像 `golang:1.26-alpine`） |
| Node.js | 22+ | 前端 / e2e（pnpm 11 已不支持 Node 18/19/20/21） |
| pnpm | 11+ | 前端 monorepo 依赖管理（`package.json` 的 `packageManager` 锁定 11.1.0） |
| PostgreSQL | 13+ | 主库（库名 `iam`），启动时由 AutoMigrate 自动建表；13+ 才能用 `DROP DATABASE ... WITH (FORCE)` |
| Redis | 5+ | SSO 会话 / 授权状态 / SLO 队列 |
| OpenTelemetry Collector（可选） | - | 链路追踪（默认 `127.0.0.1:4317`） |

---

## 2. 本地开发运行

### 2.1 后端

```bash
# 下载依赖（go.work 全模块）
make deps

# 列出可用应用
make list-apps

# 构建单个应用（产物输出到 backend/output；有效 APP 见下表）
make build APP=auth

# 运行单个应用（auth / platformadmin / tenantadmin / rpapi / gateway）
make run APP=auth
make run APP=platformadmin
make run APP=tenantadmin
make run APP=rpapi

# 单体聚合运行（一个进程挂载四者，:8100）
make run APP=gateway
```

| 应用 | 端口 | 说明 |
|---|---|---|
| auth | 8081 | 认证 + OIDC Provider（`/oidc`） |
| platformadmin | 8082 | 平台管理 |
| tenantadmin | 8083 | 租户自服务 |
| rpapi | 8084 | 面向应用的只读目录 API（`/v1/rp/directory/*`） |
| gateway | 8100 | 聚合部署（推荐日常使用） |

### 2.2 前端

```bash
cd frontend
pnpm install

pnpm dev          # platform-admin-web :4001
pnpm dev:login    # login-web :4000
pnpm dev:tenant   # tenant-admin-web :4002
```

### 2.3 启动顺序与数据准备

```mermaid
flowchart LR
    PG["启动 PostgreSQL<br/>（创建 iam 库）"] --> REDIS["启动 Redis"]
    REDIS --> BE["启动后端 gateway :8100<br/>（AutoMigrate 自动建表 + 幂等种子数据）"]
    BE --> FE["启动前端三个应用"]
    FE --> TEST["访问 platform-admin-web 验证登录"]
```

种子数据：启动时由 `pkg/seed` 幂等写入（配置 `db.seed: true` 开启）。管理员 `admin / admin123`，OAuth 客户端 `platform_admin_web` / `tenant_admin_web`（存量库的连字符编码会在启动时原地改名，保留主键）。

> **Schema 变更（列/表下线）**：本项目按新项目处理，不维护数据迁移脚本（AutoMigrate 只增不删，见 `system-design.md` §4.1）。下线列/表后，开发/测试库需**删库重建**。本地 PostgreSQL 跑在 Docker 里（容器名以 `docker ps` 为准，下例为 `postgres18`）：
>
> ```bash
> docker exec -i postgres18 psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS iam WITH (FORCE)"
> docker exec -i postgres18 psql -U postgres -d postgres -c "CREATE DATABASE iam"
> # 然后重启后端：AutoMigrate + Seed 重建全部表与种子数据
> ```
>
> `WITH (FORCE)`（PG 13+）会断开仍连着该库的会话，因此后端不必先停。旧库中残留的列/表不再被读写，属预期。确需保全旧数据时，在升级前自行执行一次性 SQL 导出/回填。
>
> **为什么本批不能只删列（存量库必须重建）**：本批把 16 个 `boolean` 列改名并改成 `varchar(16)` 枚举（如 `person.is_suspended` → `person.status`、`menu.hidden` → `hidden='enable'\|'disable'`）。AutoMigrate **不删不改**既有列，所以旧库里这些列会原样留着（新列另建、取列默认值），旧列值也不会被转换；即使手工 `ALTER COLUMN ... TYPE varchar USING bool::varchar`，得到的也是 `'true'`/`'false'`——**不是合法枚举值**，读取端按 `active`/`suspended`、`enable`/`disable` 判定会全部落到"非挂起/未启用"的默认分支。正确处置只有删库重建（上面三条命令）；确需保全业务数据时，先导出数据，再按新枚举口径人工映射后重新导入。
>
> **存量库升级到 `application.source`（2026-09-12 改造；仅在需要保全旧库数据时执行）**：`application.type` / `is_system` 与 `application_client.type` / `is_system` / `is_third_party` 已下线，归属统一收敛为 `source`（`builtin` / `first_party` / `third_party`）。AutoMigrate 只会给新列补上列默认值 `third_party`，存量行必须回填——**请在部署新代码之前执行**（新代码不再写 `type`，升级后旧列只剩列默认值，无法再据以推断）：
>
> ```sql
> -- application：is_system(内置) 优先，其次 third_party，其余归 first_party
> UPDATE application SET source = CASE
>   WHEN is_system THEN 'builtin'
>   WHEN type = 'third_party' THEN 'third_party'
>   ELSE 'first_party'
> END
> WHERE source IS DISTINCT FROM (CASE
>   WHEN is_system THEN 'builtin'
>   WHEN type = 'third_party' THEN 'third_party'
>   ELSE 'first_party'
> END);
>
> -- application_client：同规则（种子客户端 is_system=true → builtin）
> UPDATE application_client SET source = CASE
>   WHEN is_system THEN 'builtin'
>   WHEN type = 'third_party' THEN 'third_party'
>   ELSE 'first_party'
> END
> WHERE source IS DISTINCT FROM (CASE
>   WHEN is_system THEN 'builtin'
>   WHEN type = 'third_party' THEN 'third_party'
>   ELSE 'first_party'
> END);
> ```
>
> `WHERE ... IS DISTINCT FROM ...` 让脚本**幂等**（重复执行 0 行受影响），可安全重跑。只有在升级**之后**才补做时，无条件安全的只有第一步（`is_system = true` → `builtin`，它决定删除保护与租户控制台菜单范围），其余行按新策略保持 `third_party` 即可；种子数据（`platform_admin` / `tenant_admin` 与两个内置 OAuth 客户端）由 `pkg/seed` 启动时自行回填为 `builtin`，无需人工介入（归属口径见 `system-design.md` §4.3）。
>
> **存量库应用编码改为下划线连接（2026-09-12 改造）**：`application.code` 统一为下划线形态，两个内置应用 `platform-admin` → `platform_admin`、`tenant-admin` → `tenant_admin`。编码是应用的业务唯一键，改名即原地 `UPDATE`，菜单/租户订阅/角色都按 `app_id` 关联，无需一起改。**请在部署新代码之前执行**（否则种子会按新编码另建一套内置应用，旧应用仍占着旧唯一键，菜单与订阅会分裂到两套应用上）：
>
> ```sql
> UPDATE application SET code = 'platform_admin' WHERE code = 'platform-admin';
> UPDATE application SET code = 'tenant_admin'   WHERE code = 'tenant-admin';
> ```
>
> 漏做时 `pkg/seed` 也会在启动时把命中的旧编码原地改名（保留主键，幂等）；但若新旧编码**同时存在**，种子会报错中断启动要求人工确认，因此上述 SQL 是「先改名再上代码」的稳妥做法。本次只动 `application.code`，菜单编码与 OAuth `client_id` 不变。
>
> 重建后若出现登录态异常（Redis 里仍有指向已消失用户的 SSO 会话），**按前缀**清理本项目的键即可，不要 `FLUSHDB`——本地 Redis 容器常与其它项目共用：
>
> ```bash
> docker exec -i redis7 redis-cli --scan --pattern 'iam:oidc:*' | xargs -r docker exec -i redis7 redis-cli DEL
> ```
>
> **重建会让 RP 侧（Gitea / RustFS 等）的 OIDC 绑定失效——这是预期现象，不是 bug**。OIDC `sub` 为 `person:<personID>`，而 `personID` 是 `person` 表主键、随行创建（见 `sso-oidc-concepts.md` §3.4），因此同一自然人重建后 `sub` 改变：
>
> - RP 侧按 `sub` 存的账号绑定全部失效 → 表现为**重复建号**，或停在 RP 的「关联账号/绑定已有账号」页（Gitea 的典型表现）；
> - 若浏览器仍持有重建前的 IAM 会话（Redis 未清理），授权请求会用**旧** `sub` 找 person，查不到时 userinfo 只返回 `sub`，RP 会报"缺少 email/preferred_username"之类的字段缺失错误——这正是上面按前缀清 `iam:oidc:*` 的原因；
> - 处置：清 Redis 会话后用 IAM 账号重新登录一次，RP 侧重新建号或重新关联即可（本地开发无业务数据，重建绑定最省事）。

### 2.4 验证 OIDC Provider

```bash
# 服务发现
curl http://localhost:8100/oidc/.well-known/openid-configuration

# JWKS
curl http://localhost:8100/oidc/keys

# 健康检查
curl http://localhost:8100/oidc/healthz
```

---

## 3. 测试

```bash
# 运行指定应用测试（推荐；gateway 只做聚合、没有测试用例，请用 auth/platformadmin/tenantadmin/rpapi）
make test APP=auth

# 全量测试（go.work 在 backend/，且按路径逐个 use 模块）
cd backend && go test ./apps/auth/... ./apps/gateway/... ./apps/platformadmin/... ./apps/rpapi/... ./apps/tenantadmin/... ./pkg/... ./sdk/...

# 指定包 / 单个用例
cd backend && go test ./pkg/core/user/ -run TestCreate_NewPersonWithDeptRelations -v
cd backend && go test ./pkg/credential/ -run TestHashSecret_Golden -v

# 覆盖率
cd backend && go test ./apps/platformadmin/internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Lint / vet
make lint
cd backend && for m in apps/auth apps/gateway apps/platformadmin apps/rpapi apps/tenantadmin pkg sdk; do (cd $m && go vet ./...); done
```

> **为什么不能写 `go test ./...`**：`backend/go.work` 用 `use (./apps/auth … ./pkg)` 逐个列出模块，`backend/` 本身不是模块，因此 `cd backend && go test ./...` 会报 `directory prefix . does not contain modules listed in go.work`（`./apps/...` 也不行，`apps` 不是模块）。要么显式列出各模块目录，要么进入某个模块目录内跑 `./...`（`make lint` 就是用后者遍历模块）。

**单元测试约定**：auth / platformadmin / tenantadmin / rpapi 各自的 `testutil.SetupSQLite(t, entities...)` 注册内存 SQLite 为全局 iam 库，服务内 `dao.NewXxxDao()` 自动落测试库，直接断言真实 dao 行为（gateway 无 `testutil`，因为它只在 `app.go` 里挂载四个应用）。RP 侧能力另有一套不依赖 DB 的测试：`cd backend/sdk && go test ./...`（`make test-sdk`）。需要真实 PostgreSQL/Redis 的集成测试用 `pkg/testsetup`（`Initialize`/`Done`/`NewCtx`，读取 `apps/<app>/config/config.yaml`）。

**e2e（Playwright，浏览器全流程）**：

```bash
cd e2e
npm install
npx playwright install chromium
# 前置：PostgreSQL + Redis + 后端（gateway）+ 三个前端应用已启动（种子数据启动时自动写入）
npx playwright test
```

覆盖场景：首次登录、SSO 免密、登出即失效、双向 SSO、全局登出（SLO）、Cookie 隔离等（详见 `e2e/README.md`）。

---

## 4. Docker 部署

```bash
# 构建镜像（推荐 gateway 单体）。镜像名为 <app>:<tag>，tag = <最近 git tag>-<short commit>，无 tag 时仅 <short commit>
make docker-build APP=gateway

# 运行容器：PORT 必须等于该应用的监听端口（默认 8099，与容器内端口不一致，会连不上）
make docker-run APP=gateway PORT=8100
make docker-run APP=auth    PORT=8081
```

镜像构成：Dockerfile 位于 `backend/apps/<app>/scripts/Dockerfile`，构建上下文为 `backend/`；构建阶段 `golang:1.26-alpine`、运行阶段 `alpine:3.22`；容器名即 `$(APP)`（同名容器会先被删除）；`EXPOSE` 对应各应用端口。

生产容器注意：

- 通过环境变量 `APP_CONFIG_PATH` 挂载生产 `config.yaml`（含正式 issuer、密钥、`cookieSecure: true`）；
- **默认镜像内嵌仓库里的 dev 签名私钥与 dev 加密口令**（Dockerfile 会 `COPY config/oidc-dev-key.pem`，`config.yaml` 内嵌 dev `encryptionKey`）：生产上线前必须替换为外部 Secret 挂载并覆盖这两项；
- 签名私钥与加密密钥通过 Secret 挂载，不写入生产镜像；
- 数据库、Redis 使用托管实例，与容器网络隔离。

---

## 5. 多环境部署拓扑

```mermaid
flowchart TB
    subgraph PROD["生产环境"]
        LB["负载均衡 / 网关"]
        subgraph AUTH_CLUS["auth 集群（多副本）"]
            A1["auth-1 :8081"]
            A2["auth-2 :8081"]
        end
        subgraph PLAT_CLUS["platformadmin 集群"]
            P1["platformadmin-1 :8082"]
        end
        subgraph TEN_CLUS["tenantadmin 集群"]
            T1["tenantadmin-1 :8083"]
        end
        subgraph RP_CLUS["rpapi 集群"]
            R1["rpapi-1 :8084"]
        end
        REDIS_SHARED[("Redis（认证共享）<br/>SSO 会话/授权状态/SLO 队列")]
        PG[("PostgreSQL")]
    end
    LB --> AUTH_CLUS
    LB --> PLAT_CLUS
    LB --> TEN_CLUS
    LB --> RP_CLUS
    AUTH_CLUS --> REDIS_SHARED
    PLAT_CLUS --> REDIS_SHARED
    TEN_CLUS --> REDIS_SHARED
    AUTH_CLUS --> PG
    PLAT_CLUS --> PG
    TEN_CLUS --> PG
```

| 部署形态 | 适用 | 说明 |
|---|---|---|
| **单体聚合**（gateway 单进程） | 小规模/开发 | 一个进程挂载四应用，端口 8100 |
| **分体部署** | 中大规模 | auth / platformadmin / tenantadmin / rpapi 独立进程独立扩缩容 |
| **auth 多副本** | 高可用 | 必须共享同一认证 Redis（会话/授权状态），PostgreSQL 主库；`/oidc` 无状态化依赖 Redis |

**关键约束**：

- 启用了 `enableSSOSessionValidation` 的所有应用必须**共享同一认证 Redis**；
- `issuer` 必须与最终访问入口一致（负载均衡后仍应是客户端可见的正式域名）；
- 签名密钥在多副本间保持一致（同一文件/同一 PEM 配置），避免 kid 漂移。

---

## 6. 常见排障

| 现象 | 排查 |
|---|---|
| 登录后接口 401 | 检查 iss/aud 校验配置、共享 Redis、SSO 会话活性开关（`enableSSOSessionValidation`） |
| `/oidc` 返回 404 | 确认访问的是 auth（:8081）或 gateway（:8100），且 issuer 前缀为 `/oidc` |
| 生产启动失败 | 检查签名/加密密钥是否显式配置（fail-closed）；`cookieSecure` 是否与 HTTPS 匹配 |
| SSO 免密失效 | 检查 `iam_sso_session` Cookie 是否写入（Domain/SameSite/Secure）、Redis 会话是否存在 |
| 跨站无法共享 SSO | 检查 `cookieSameSite: none` + `cookieSecure: true` |
| e2e 失败 | 按 `e2e/README.md` 核对服务映射与种子数据 |
| 令牌验签失败 | 对比 `/oidc/keys` 与 RP 配置公钥是否一致（kid/密钥内容） |
