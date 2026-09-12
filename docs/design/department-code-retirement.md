# 部门业务编码下线（`department.code` 移除）

> 状态：**已落地**（2026-09-12）
> 决策：部门不需要业务编码，`department.code` 全链路下线（不留别名、不加兼容层）
> 关联：角色侧的同类下线见 [`role-code-retirement.md`](./role-code-retirement.md)；本文修订 `department-container-rename.md` 中 `name`/`code` 并列表述
> 影响一句话：1 列、1 DAO 条件项、1 个共享 object 字段、1 处种子赋值、1 个前端表单项、4 个前端文件、1 份 swagger 生成物。

---

## 1. 为什么下线

`department.code` 原注释为「部门编码(租户内唯一,可空,外部系统同步用)」，但这份语义从未真正成立：

- **没有唯一性约束**：既无 DB 唯一索引，也无 service 校验，`(tenant_id, code)` 重复可写入——与注释承诺的「租户内唯一」不符。
- **没有任何读取路径**：`DepartmentCond.Code` 全仓无调用点，部门树/子部门/成员等一切查询都按 `name`/`status`/`parent_id` 过滤；DTO 里它只是一个可空的透传字段。
- **唯一写入点是两条"顺手带上"的赋值**：`seedRootDepartment` 把 `tenant.Code` 抄进根部门，`departmentSvc.Create` 把请求里的值原样落库。
- **前端只是个可空输入框**：「部门编码（可空，外部系统同步用）」，没有列、没有搜索、编辑时回显一次即弃。

也就是说它是一个「占位字段」：留着就要在 model/DAO/object/DTO/前端表单/文档里各维护一份，删掉零功能损失。真正的部门标识是 `department.id`（UUID v7），人读标识是 `department.name`。

> 与角色不同：角色的 `code` 曾承载「应用内唯一」约束与内置角色定位，因此下线时把唯一性移交给了 `name`；
> 部门的 `code` **从未被约束、也从未被查询**，所以本次只做删除，**没有**给部门名称新增唯一性校验（部门树允许同名节点，这是既有行为，本次不变）。

## 2. 变更清单

**后端**

| 位置 | 变更 |
|---|---|
| `pkg/iam/model/department.go` | 删除 `Code` 字段（`code` 列） |
| `pkg/iam/dao/department.go` | `DepartmentCond` 删除 `Code` 及对应 BuildCondition 分支（删除前已无调用点） |
| `pkg/iam/object/objtenant/department.go` | `DepartmentBaseInfo` 删除 `Code`（同时影响创建/更新请求与树/子节点响应） |
| `apps/tenantadmin/internal/service/svctenant/department.go` | Create 不再落 `code`；Tree / Children 不再回填 `code` |
| `pkg/seed/seed.go` | `seedRootDepartment` 不再把 `tenant.Code` 抄进根部门（根部门只继承租户名） |

**前端**

| 位置 | 变更 |
|---|---|
| `packages/types/src/department.ts` | `DepartmentItem` / `DepartmentChildItem` 删除 `code` |
| `apps/tenant-admin-web/src/api/department.ts` | `createDepartment` / `updateDepartment` 入参删除 `code` |
| `apps/tenant-admin-web/src/pages/department/index.tsx` | 删除「部门编码」表单项、编辑回显与 `editingNode` 类型里的 `code` |

**文档 / 生成物**

- `docs/design/system-design.md`（`department` 表 ER）、`department-container-rename.md`（追加字段下线说明）同步。
- `backend/apps/tenantadmin/docs/tenantadmin_docs.go` 由 `make swag APP=tenantadmin` 重新生成。

## 3. 数据库处置

按 `AGENTS.md`「按新项目处理」约定：**不写迁移脚本、不加兼容旧库的分支**。开发/测试库删库重建后由 `AutoMigrate` + `Seed` 产出目标结构（旧库残留 `department.code` 列不再被读写）。

## 4. 验收

| 验证 | 命令 | 结果 |
|---|---|---|
| 后端编译 | 5 模块 `go build ./...` / `go vet ./...` | 通过 |
| 后端测试 | `pkg`、`apps/tenantadmin`、`apps/auth`、`apps/platformadmin` `go test ./...` | 全绿 |
| 前端 | `pnpm typecheck`；`pnpm build:tenant`；`pnpm test:all` | 通过 |
| Swagger | `make swag APP=tenantadmin` | 重新生成，0 处「部门编码」 |
| 零残留 | `grep -rn "部门编码\|department\.code\|dept_code" backend frontend docs/design` | 仅本文件与历史设计文档的说明性引用 |
