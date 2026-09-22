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
| PostgreSQL | 13+ | 主库（库名 `iam`），启动时由 AutoMigrate 自动建表（**只建表、不写数据**）；13+ 才能用 `DROP DATABASE ... WITH (FORCE)` |
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
    REDIS --> BE["启动后端 gateway :8100<br/>（AutoMigrate 自动建表，不写任何数据）"]
    BE --> FE["启动前端三个应用<br/>（login-web :4000）"]
    FE --> INSTALL["打开 login-web 的 /install<br/>填入 BOOTSTRAP_TOKEN 完成首次初始化"]
    INSTALL --> TEST["访问 platform-admin-web :4001 验证登录"]
```

**数据准备只有两种来源**：schema 由启动时的 `AutoMigrate` 建出（只增不删），**内置数据由初始化页面一次性写入**——启动期不写任何数据，`pkg/seed` 的唯一写通道 `Bootstrap` 只有 `POST /install/initialize` 一个调用方（详见 §2.4）。因此**全新库的启动顺序必须是「先起后端（带 `BOOTSTRAP_TOKEN`）→ 再打开安装页完成引导」**；未初始化期间业务端点会被 `pkg/middleware.BootstrapGuard` 拦成 HTTP 409（错误码 `107004`），这是预期行为，不是服务故障。

> **为什么把播种从启动期挪到安装页**：启动期播种必须"每次启动都把内置行收敛到代码定义"，于是应用既要建表又要当数据写者，运维在控制台改过的内置字段会在下次重启被悄悄收回（双写者），跨版本改名也只能靠启动分支打补丁。改成一次性引导后，写者唯一、时机明确（部署时由人确认），应用启动退化为纯只读。代价已经明确接受：内置数据的后续调整全部由运维在控制台完成，不再自动下发（见 §2.5）。

> **Schema 变更（列/表下线）**：本项目按新项目处理，不维护数据迁移脚本（AutoMigrate 只增不删，见 `system-design.md` §4.1）。下线列/表后，开发/测试库需**删库重建**。本地 PostgreSQL 跑在 Docker 里（容器名以 `docker ps` 为准，下例为 `postgres18`）：
>
> ```bash
> docker exec -i postgres18 psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS iam WITH (FORCE)"
> docker exec -i postgres18 psql -U postgres -d postgres -c "CREATE DATABASE iam"
> # 然后重启后端：AutoMigrate 重建全部表；再打开 /install 重新完成一次初始化
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
> -- application_client：同规则（is_system=true 的内置客户端 → builtin）
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
> `WHERE ... IS DISTINCT FROM ...` 让脚本**幂等**（重复执行 0 行受影响），可安全重跑。只有在升级**之后**才补做时，无条件安全的只有第一步（`is_system = true` → `builtin`，它决定删除保护与租户控制台菜单范围），其余行按新策略保持 `third_party` 即可。**新代码不会再自动回填 `source`**：内置应用与两个内置 OAuth 客户端的 `source=builtin` 由 L1 首次引导在**创建时**写入，启动期不做任何回写；存量库必须靠上面的 SQL 自己改对，漏做不会被修复（归属口径见 `system-design.md` §4.3）。
>
> **存量库应用编码改为下划线连接（2026-09-12 改造；仅在需要保全旧库数据时执行）**：`application.code` 统一为下划线形态，两个内置应用 `platform-admin` → `platform_admin`、`tenant-admin` → `tenant_admin`。编码是应用的业务唯一键，改名即原地 `UPDATE`，菜单/租户应用/角色都按 `app_id` 关联，无需一起改。**请在部署新代码之前执行**——新代码里**没有任何**自动改名分支（旧实现的 legacy 原地改名已随启动期播种一并删除）：漏做时旧编码会原样留着，若之后删库重建，L1 首次引导会按新编码另建一套内置应用，菜单与租户应用将分裂到两套应用上。
>
> ```sql
> UPDATE application SET code = 'platform_admin' WHERE code = 'platform-admin';
> UPDATE application SET code = 'tenant_admin'   WHERE code = 'tenant-admin';
> ```
>
> 本次只动 `application.code`，菜单编码与 OAuth `client_id` 不变。
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

### 2.4 首次初始化（安装页引导）

全新库（或按 §2.3 重建后的库）在第一次可用之前必须完成一次初始化：schema 由后端启动时的 `AutoMigrate` 建好，内置数据（平台租户、根部门、两个内置应用与 OAuth 客户端、内置菜单与角色、内置管理员）则由安装页**一次性**写入。这一步必须由部署人员本人完成——不能用脚本绕过，也没有任何"镜像内置默认口令"可用。

1. **准备数据库**：目标库为空；`AutoMigrate` 会在后端启动时自动建表，不需要手工建表，也不需要（也不应该）预置任何数据。
2. **带一次性初始化令牌启动后端**：

   ```bash
   # 令牌是部署期的一次性机密，建议用长随机串；不要写进 config.yaml，也不要进镜像
   export BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
   make run APP=gateway
   ```

   > `BOOTSTRAP_TOKEN` 只从**环境变量**读取。**未设置时 `/install/initialize` 整体不可用**：`GET /install/status` 回 `tokenRequired=false`，提交返回 HTTP 503（错误码 `107002`）。这是 fail-closed 设计——宁可"没人能初始化"，也不能让一个没有令牌保护的写接口在公网裸奔。

3. **打开安装页**：先启动登录门户（`cd frontend && pnpm dev:login`，:4000），浏览器访问 `http://localhost:4000/install`；未初始化时访问 `http://localhost:4000/login` 也会被 `InstallGuard` 自动跳到这里。页面共三步，但**只有第三步发一次写请求**（`POST /install/initialize`，其余校验都在浏览器内完成）：

   1. **租户**：平台租户名称（默认「平台运营中心」，留空由后端取默认值）；
   2. **管理员**：初始管理员用户名、密码、确认密码、姓名、邮箱 / 手机号（邮箱与手机号**至少填一个**）；
   3. **内置数据确认**：核对将写入的内置数据，并粘贴 `BOOTSTRAP_TOKEN` 的值（作为请求头 `X-Bootstrap-Token` 发送，不写入浏览器存储）。

   管理员口令**由运维在这里自己设定**：后端不预置任何默认口令（历史实现里的 `credential.BootstrapAdminPassword` = `admin123` 已删除），服务端按 `credential.ValidateStrength` 校验 **8–128 位且同时包含大写字母、小写字母与数字**，不满足当场拒绝（HTTP 400，错误码 `107006`）。内置控制台的 OIDC 回调地址来自配置 `oidc.consoles`（留空则用内置默认值，见 `configuration-reference.md`）。

4. **确认提交**：后端在**单个事务**内写入全部内置数据，返回管理员用户名、登录地址与控制台入口；页面提示"内置数据已写入，后续启动不会再重复写入"。之后即可用刚设定的账号登录 platform-admin-web（:4001）/ tenant-admin-web（:4002）。

**安全约束（部署必读）**：

- **`BOOTSTRAP_TOKEN` 是部署期一次性机密**：只在引导阶段设置，初始化完成后应立即从运行环境移除；它不该长期留在生产环境变量里。
- **`/install` 只应在受信网络内可达**：它不需要登录态，唯一门禁就是这个令牌。若部署机在公网可达，推荐用 **SSH 端口转发**在本地打开，不要把安装页暴露到公网：

  ```bash
  # 在本地机器执行：把部署机的 login-web 与后端端口映射到本地，无需对外暴露 /install
  ssh -N -L 4000:127.0.0.1:4000 -L 8100:127.0.0.1:8100 deploy@<host>
  # 然后本地浏览器访问 http://localhost:4000/install
  ```

  完成初始化后关闭转发；不要在负载均衡 / 反向代理上长期暴露 `/install/*`。
- **端点会永久自锁**：初始化成功后再次调用 `/install/initialize` 一律返回 HTTP 409（错误码 `107000`），重启、换副本都不会解除；`GET /install/status`（公开、只读）回 `initialized=true`，可给部署脚本或监控用来确认状态。
- **未初始化期间业务端点不可用是预期行为**：`pkg/middleware.BootstrapGuard` 会把业务端点拦成 HTTP 409（错误码 `107004`），只放行 `/install` 与 `/oidc` 的健康检查 / 服务发现 / 登出端点。集成方在初始化前探测收到 409，属正常引导态，不是服务故障。

### 2.5 版本升级时的菜单变更清单

**菜单行归运维**：L1 首次初始化只写入一次内置菜单，之后菜单由「菜单管理」页维护（可新增根菜单/子菜单，可删除任意菜单）。因此**版本升级带来的新菜单不会被自动下发**，升级后需要人工补录。

```bash
# 打印内置菜单的完整 15 字段清单（Markdown 表格）
make print-builtin-menus
```

用法：

1. 升级前先跑一次，留存本次输出的旧清单；
2. 升级后再跑一次，`diff` 两份输出——**新增的行就是需要手工补录的菜单**；
3. 在对应控制台的「菜单管理」页按清单录入：先建 `type=directory` 的父级，再建子菜单（`parentCode` 指向父级的 `code`）；
4. 清单里后 5 列（`redirect` / `hidden` / `externalLink` / `keepAlive` / `status`）是**建表列默认值**，照填即与首次初始化产物一致；
5. 新菜单若属于某个角色的默认可见范围，再去「角色管理」页补授权（目录菜单不需要授权）。

> **不要复用已下线菜单的 `code`**：首次初始化引导按不可见的 `seed_key` 认行，而 `seed_key` 只在创建时写入、之后不变；且库一旦初始化，引导永久自锁，代码里改了 `code` 既不会改动已有行、也不会补建新行。升级新增菜单请用新的 `code`。
>
> **漏补菜单的后果是"导航里看不到"，不是"接口 403"**：菜单不参与 API 鉴权（权限由角色与字段权威决定）。因此漏补只影响可用性，不会造成越权，可以从容补录。
>
> 历史版本曾用「退役菜单清单（`retiredMenus`）」在启动时自动清理已下线的内置菜单。**该机制已删除**：菜单的下线与调整现在完全由运维在控制台完成（详见 `system-design.md` §4.5）。

### 2.6 验证 OIDC Provider

```bash
# 初始化状态（公开只读；initialized=false 说明还没走 §2.4 的安装页）
curl http://localhost:8100/install/status

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
# 前置：PostgreSQL + Redis + 后端（gateway，带 BOOTSTRAP_TOKEN）+ 三个前端应用已启动，
#       且已由 global-setup 通过 /install/initialize 完成首次初始化（见 e2e/README.md）
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
- **首次部署额外注入一次性 `BOOTSTRAP_TOKEN`**（建议用 Secret 而非明文 env），用它在 `/install` 完成初始化后立即移除；`/install` 不要经负载均衡暴露到公网，运维用 SSH 端口转发打开（见 §2.4）；
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
- 签名密钥在多副本间保持一致（同一文件/同一 PEM 配置），避免 kid 漂移；
- **初始化只需在一个副本上做一次**：是否已初始化以库内平台租户行为准（`seed.IsInitialized`），`BootstrapGuard` 每次探测直接查库，没有进程内状态，因此初始化完成后所有副本在下一次探测即整体放行业务端点；`BOOTSTRAP_TOKEN` 只需在引导阶段为"接流量的那一侧"设置；
- **多副本同时提交初始化是安全的**：`seed.Bootstrap` 用 Postgres 事务级 advisory lock 串行化，只有一个副本真正写入，其余得到 `already_initialized`（HTTP 409），不会写出两套内置数据；
- **`oidc.consoles` 的回调地址必须与最终访问入口一致**：它在 L1 首次引导时写入内置 OAuth 客户端，之后改配置不会回写（`redirect_uris` 归运维、可在控制台改）；分体部署时 `backChannelLogoutURI` 要显式配置，否则只能按 `issuer` 派生。

---

## 6. 常见排障

| 现象 | 排查 |
|---|---|
| 登录后接口 401 | 检查 iss/aud 校验配置、共享 Redis、SSO 会话活性开关（`enableSSOSessionValidation`） |
| `/oidc` 返回 404 | 确认访问的是 auth（:8081）或 gateway（:8100），且 issuer 前缀为 `/oidc` |
| 生产启动失败 | 检查签名/加密密钥是否显式配置（fail-closed）；`cookieSecure` 是否与 HTTPS 匹配 |
| SSO 免密失效 | 检查 `iam_sso_session` Cookie 是否写入（Domain/SameSite/Secure）、Redis 会话是否存在 |
| 跨站无法共享 SSO | 检查 `cookieSameSite: none` + `cookieSecure: true` |
| e2e 失败 | 按 `e2e/README.md` 核对服务映射与首次初始化（e2e 由 `global-setup` 调 `/install/initialize`，需后端带 `BOOTSTRAP_TOKEN`） |
| 业务接口一律 409 | 库还没初始化（错误码 `107004`）：按 §2.4 打开安装页完成引导 |
| `/install/initialize` 返回 503 | 后端未设置 `BOOTSTRAP_TOKEN` 环境变量（错误码 `107002`），或未初始化探测超时 |
| `/install/initialize` 返回 409 | 库已初始化且端点永久自锁（错误码 `107000`）；需要重来只能删库重建（§2.3） |
| 令牌验签失败 | 对比 `/oidc/keys` 与 RP 配置公钥是否一致（kid/密钥内容） |
