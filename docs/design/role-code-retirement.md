# 角色业务编码下线（`role.code` 移除）

> 状态：**已落地**（2026-09-12）
> 决策：角色不需要业务编码，`role.code` 全链路下线（不留别名、不加兼容层）
> 取代/修订：`role-dimension-redesign.md` 中「内置角色保护仅锁定 `code`」的表述（该文其余内容仍按原样作为历史记录）
> 影响一句话：1 列、1 内置角色定位常量、1 错误码、约 10 个后端文件、4 个前端文件、1 份 swagger 生成物。

---

## 1. 为什么下线

角色本来就有一个**系统身份**（`role.id`，UUID v7 字符串主键）和一个**人读标识**（`role.name`）。`code` 是第三个标识，且：

- **前端从来没有真正使用它**：角色列表、关键词搜索、`RoleAssignEditor` 的角色下拉都只需名称；编码列只是把内部命名暴露给租户。
- **它带来真实的维护成本**：创建/编辑必填、`(tenant_id, app_id, code)` 唯一性校验、内置角色的「禁改编码」守卫、种子与权限开通都要维护 `admin` / `tenant_admin` 两个魔法字符串（`ProvisionRoleCode`）。
- **它诱导错误建模**：把「内置角色识别」这种系统语义寄托在业务编码上，改名/改码就会失联。

结论：**角色不需要编码**。一个实体的标识由主键承担，人读标识由名称承担。

## 2. 替代语义（单一事实源）

| 原语义 | 现语义 |
|---|---|
| 角色标识 | `role.id`（UUID v7，不变） |
| 应用内可读标识 + 唯一性约束 | `role.name`：`(tenant_id, app_id, name)` 应用内唯一（承接原「编码应用内唯一」的约束，避免同一应用下出现两个同名角色而无法在授权下拉中区分） |
| 内置角色定位（原 `role.code = tenant_admin`） | `(tenant_id, app_id, source=builtin)`；内置管理员能力判定仍是 `source=builtin && admin_type=admin`（`RoleEntity.IsBuiltinAdmin`） |
| 内置角色核心字段保护 | 无（`code` 已不存在）；内置角色仍**禁止删除**（`RoleDeleteBuiltinForbiddenError`），`admin_type` 仍由种子 / 权限开通幂等回填 |

## 3. 变更清单

**后端**

| 位置 | 变更 |
|---|---|
| `pkg/iam/model/role.go` | 删除 `Code` 字段（`code` 列） |
| `pkg/iam/dao/role.go` | `RoleCond` 删除 `Code`；`Keyword` 由「名称 LIKE OR 编码 LIKE」收敛为「名称 LIKE」 |
| `apps/tenantadmin/internal/dto/dtotenant/role_request.go` | `RoleCreateReq`/`RoleUpdateReq` 删除 `Code`；`Name` 注释改为「应用内唯一」 |
| `apps/tenantadmin/internal/dto/dtotenant/{role_response.go,user_response.go}` | `RolePageListItem`/`RoleDetailResp`/`UserRoleItem` 删除 `Code` |
| `apps/tenantadmin/internal/service/svctenant/role.go` | Create 的唯一性校验由 code 改为 name（仍限 `tenant_id + app_id`）；Update 删除「内置角色禁改核心字段」守卫；Detail/PageList 不再回填 code |
| `apps/tenantadmin/internal/service/svctenant/user.go` | `UserRoleItem` 不再回填 code |
| `pkg/iam/tenant/provision.go` | 删除 `ProvisionRoleCode` 常量；`ensureBuiltinRole` 改按 `(tenant_id, app_id, source=builtin)` 幂等定位，只回填 `admin_type` |
| `pkg/seed/seed.go` | 删除 `seedRole.code`；`seedRoles` 返回单个内置角色（原按 code 建 map）；`seedRoleMenus`/`seedAdminUserRole` 直接接收角色实体 |
| `pkg/code/permission.go` | 删除 `RoleUpdateBuiltinForbiddenError`（100707）及其文案（唯一用途是拦截改 `code`） |

**前端**

| 位置 | 变更 |
|---|---|
| `packages/types/src/tenant.ts` | `TenantRoleItem`/`TenantRoleCreateReq`/`TenantUserRoleItem` 删除 `code` |
| `apps/tenant-admin-web/src/api/role.ts` | `updateTenantRole` 入参删除 `code` |
| `apps/tenant-admin-web/src/pages/role/index.tsx` | 删除「角色编码」列与表单项、编辑回显、搜索占位文案；表单改在名称上标注「应用内唯一」 |
| `apps/tenant-admin-web/src/components/RoleAssignEditor.tsx` | 角色下拉文案由 `名称（编码）` 收敛为 `名称` |

**文档 / 生成物**

- `docs/design/api-reference.md`（角色接口、建租户管理员约定）、`system-design.md`（`role` 表 ER）、`glossary.md`（角色词条）、`tenant-admin-provisioning-design-20260912.md`（角色 upsert 键）同步。
- `backend/apps/tenantadmin/docs/tenantadmin_docs.go` 由 `make swag APP=tenantadmin` 重新生成。

## 4. 数据库处置

按 `AGENTS.md`「按新项目处理」约定：**不写迁移脚本、不在启动流程加兼容分支**。开发/测试库删库重建后由 `AutoMigrate` + `Seed` 产出目标结构（旧库残留 `role.code` 列不被读写）。因角色业务编码已删除，**无法再按 code 回填存量内置角色的 `source`/`admin_type`** —— 旧库按上述方式重建即可。

## 5. 验收

| 验证 | 命令 | 结果 |
|---|---|---|
| 后端编译（含测试） | 5 模块 `go vet ./...` | 通过 |
| 后端测试 | 5 模块 `go test ./...` | 全绿 |
| 前端 | `pnpm typecheck`；`pnpm build:tenant`；`pnpm test:all` | 通过（34 tests） |
| Swagger | `make swag APP=tenantadmin` | 重新生成，0 处「角色编码」 |
| 零残留 | `grep -rn "role.code\|ProvisionRoleCode\|角色编码" backend frontend/packages frontend/apps docs/design` | 仅本文件与历史设计文档的说明性引用 |
