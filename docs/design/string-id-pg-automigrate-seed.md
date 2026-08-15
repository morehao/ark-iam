# 字符串主键与启动自动化设计（string-id / PG / AutoMigrate / Seed）

> 状态：已落地（feat/pg-string-id 分支）
> 涉及：全部 30 张数据表主键改为字符串、数据库由 MySQL 切换 PostgreSQL、启动自动建表与幂等种子数据。

## 1. 主键选型：UUID v7（时间有序）

### 1.1 背景

原 schema 全部主键为 `BIGINT UNSIGNED AUTO_INCREMENT`。改造目标：**所有表主键改为字符串类型**，
主键生成由 `golib` 提供（`github.com/morehao/golib/dbaccess/gormdao` 的 `BaseEntity`）。

### 1.2 golib 的生成方式

`gormdao.BaseEntity`（语义等同 `gorm.Model`，主键为 `varchar(36)`）：

```go
type BaseEntity struct {
    StringID                       // ID string `gorm:"column:id;primaryKey;type:varchar(36)"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt
}
// BeforeCreate 在插入前自动生成主键 ID（UUID v7，时间有序，主键索引友好）；
// 已显式赋值的 ID 不会被覆盖。
func (s *StringID) BeforeCreate(tx *gorm.DB) error {
    if s.ID == "" {
        s.ID = gutil.GenUUID() // uuid.Must(uuid.NewV7()).String()
    }
    return nil
}
```

项目中全部实体由 `gorm.Model` 改为内嵌 `gormdao.BaseEntity`，所有 ID/外键列（`tenant_id`、`person_id`、
`created_by` 等）由 `uint` 改为 `string`，DAO 泛型 ID 由 `uint` 改为 `string`，联动改造
DTO/Service/Controller/claims/SSO/测试与前端类型包。

### 1.3 主流生成方案对比（调研结论）

| 方案 | 位数/格式 | 时间有序 | 索引友好 | 分布式 | 说明 |
|---|---|---|---|---|---|
| **UUID v7** | 128bit，36 字符 | ✅（毫秒时间戳前缀） | ✅ 顺序写 | ✅ 无需协调 | **现代主流推荐**；PG 18 已内置 `uuidv7()`；golib 即采用 |
| UUID v4 | 128bit，36 字符 | ❌ 随机 | ❌ B-tree 页分裂 | ✅ | 早期事实标准，索引膨胀问题明显 |
| Snowflake | 64bit int | ✅ | ✅ | ⚠️ 需 worker ID 分配 | 社区成熟但引入时钟回拨/worker 管理复杂度 |
| ULID | 128bit，26 字符 | ✅ | ✅ | ✅ | 可排序、可读性好，生态略逊 |
| KSUID / xid | 160/96bit | 部分 | 部分 | ✅ | 各自场景小众 |
| 数据库自增 | 64bit int | ✅ | ✅ | ❌ 单点/分库困难 | 原方案；多租户分库、数据迁移、ID 防枚举场景不友好 |

**结论**：UUID v7 在"时间有序（主键索引友好）+ 无需分布式协调 + 不可枚举 + 跨库稳定"上综合最优，
与 golib 内置生成方式一致，故直接采用 golib `gormdao.BaseEntity` + `gutil.GenUUID()`（UUID v7），
不额外引入 Snowflake/ULID 依赖。

## 2. 数据库切换 PostgreSQL

- 驱动：`github.com/morehao/golib/dbaccess/dbgorm/driver/postgres`（`dbclient` 引入替换 mysql 驱动）。
- 连接串：`postgres://user:pass@127.0.0.1:5432/iam?sslmode=disable&TimeZone=Asia/Shanghai`
  （四个应用 `config/config.yaml` 已更新）。
- 列类型：移除 MySQL 专有类型（`bigint unsigned`/`tinyint(1)`/`datetime`）——
  时间列不再显式声明类型（PG→`timestamptz`，SQLite→`datetime`，由 gorm 按方言自动映射）；
  布尔/枚举类 int8 用 `smallint`；ID 列 `varchar(36)`。
- 单元测试仍用内存 SQLite（gorm 方言抽象，测试不依赖 PG）。

> ⚠️ 经验 1：gorm 的 `default X`（空格形式）标签不会被识别为默认值（`ParseTagSetting` 按 `:` 切分），
> 会导致 `not null` 列插入 NULL 报错。本项目已统一为 `default:X` 冒号形式
> （`default:CURRENT_TIMESTAMP`、`default:('{}')` 等），并在 PG 上实测验证。
>
> ⚠️ 经验 2：`user` 是 PostgreSQL 保留字。DAO 层按 `表名.列名` 拼 SQL（gormdao `deletedScope` 及各 Cond），
> 生成未加引号的 `user.deleted_at` 在 PG 上报 `syntax error at or near "."`（登录链路即触发）。
> 已把用户表改名为 **`tenant_user`**（`model.TableNameUser`，与 `person` 自然人语义区分更清晰），其余 29 张表名均非保留字。
>
> ⚠️ 经验 3：golib `gormplugin.ScopePlugin`（多租户过滤）生成 **MySQL 反引号**限定符
> （`` `tenant_user`.tenant_id = ? ``），在 PG 上报 `syntax error at or near "."`——
> 任何带 tenant_id 上下文（登录后携带 token 的请求）查询被过滤的表都会触发。
> 已在 `pkg/dbclient` 自建 PG 兼容插件（双引号限定符，逻辑与 golib 一致），
> 并新增 `//go:build pg` 回归测试 `pkg/dbclient/tenant_scope_pg_test.go`。

## 3. 启动自动建表（GORM AutoMigrate）

`pkg/iam/model/automigrate.go`：

```go
func AutoMigrateAll(db *gorm.DB) error {
    return db.AutoMigrate(AllEntities()...) // 30 张表
}
```

- 幂等：只新增缺失的表/列/索引，不删不改既有结构，可安全重复执行与多实例并发。
- 开关：`db.auto_migrate`（各应用 `config/config.yaml`，默认 `true`）。
- 各应用 `cmd/init.go` 在 `InitMultiDB` 后调用。
- 取代 `backend/scripts/sql/iam_schema.sql`（MySQL 方言，已废弃）。

## 4. 种子数据：启动时幂等写入

`pkg/seed/seed.go`：以 Go 代码取代 `scripts/sql/iam_seed_data.sql`（MySQL 方言）。

- **写入内容**：平台租户、顶级部门、应用（admin/tenant-admin）、角色（admin/user/guest）、
  资源与 14 个 scope、角色-scope 关联、18 个菜单、角色-menu 关联、租户应用订阅、
  管理员账号（`admin / admin123`）、两个 OIDC 测试客户端。
- **幂等策略**：按唯一键查重（租户 `code`、应用 `code`、角色 `(tenant_id, code)`、
  scope `(tenant_id, name)`、菜单 `(app_id, code)`、person `username`、client `client_id`、
  关联表 `(tenant_id, role_id, scope_id)` 等），已存在则跳过、不存在则创建；
  主键由 `BeforeCreate` 生成 UUID v7，实体间关联动态接线，**不依赖固定数字 ID**。
- **开关**：`db.seed`（默认 `true`），启动时在 AutoMigrate 之后执行。
- 已在 PostgreSQL 17 实测：首次写入 + 二次执行幂等校验全部通过（PG 集成测试
  `pkg/seed/seed_pg_test.go`，`//go:build pg` 标签，不参与常规 CI）。

## 5. 影响面与迁移注意

- 后端：`pkg/iam/model`（30 实体）、`pkg/iam/dao`（泛型 ID）、`pkg/iam/object`、三应用
  DTO/Service/Controller、`objauth.TokenClaims`（tenant_id/user_id 为字符串）、
  SSO/SLO（personID 字符串）、OIDC subject（`person:<uuid>`）、API Key、测试（SQLite 自足化）。
- 前端：类型包 `packages/types`、`packages/api` 及三个 web 应用 ID 字段 `number → string`。
- 存量 MySQL 数据迁移：需按新主键类型重建数据（无自动转换脚本）；
  建议直接以 AutoMigrate + Seed 初始化全新 PG 库。
