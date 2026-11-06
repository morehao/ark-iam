# 待办清单（TODO / Backlog）

> 本文记录**尚未实施、但已明确方向**的待办项。与 `docs/design/` 的关系：
> `docs/design/` 记录「已落地且与代码一致」的设计事实；本文记录「还没做、但已经知道要怎么做」的事项。
> 某项落地后，结论收敛进 `docs/design/` 对应文档，并从本文移除（或标记为已完成）。
>
> 维护约定：每项必须有「目标 / 要改的文件 / 验收标准」三要素，便于直接接手执行。

---

## 目录

1. [P0 前端组件发布为 npm 包（**已完成**）](#1-p0-前端组件发布为-npm-包已完成---2026-11-06)
2. [P1 前端门面包 `@morehao/ark-iam-react`](#2-p1-前端门面包-ark-iamreact)
3. [P2 管理台前端的内嵌/挂载](#3-p2-管理台前端的内嵌挂载)
4. [P3 后端相关待办](#4-p3-后端相关待办)
5. [已完成的记录](#5-已完成的记录)

---

## 1. P0 前端组件发布为 npm 包（**已完成 — 2026-11-06**）

> **状态：已落地（最终采用「单门面包」形态）。** 对外**只发布一个包** `@morehao/ark-iam-react`：它把内部 `@ark-iam/{types,api,auth,ui}` 的代码与类型**打进自身 dist**，以 ESM 子路径 `@morehao/ark-iam-react/{types,api,auth,ui}` 按需暴露，根入口 `@morehao/ark-iam-react` 一把梭。T1~T9 全绿：库构建（ESM + `.d.ts`，含类型自包含 bundle）、入口 `exports` 指向 `dist`、`publishConfig.access=public`、peer/dev 依赖分类、脚本化版本/发布通道（仅门面包）、接入文档、仓外空项目冒烟（tarball 离线 → 外部 vite+react 项目 `actionColumn` 表格渲染 + `createAuthProvider` 登录跳转 PKCE 成功；独立 facade 消费测试：根入口 + 四子路径 runtime/tsc 均过）均通过。
> 运行期配置优先级：**props > `/config.json` > `VITE_*` > 相对默认值**（`/config.json` 由后端 `iam/app_frontend.go` 从 `Config.Raw` 与请求 Host 派生，含 CORS）；仓内 dev 走 `src`（`frontend/vite.alias.ts` 的 `arkIamAlias`），发布走 `dist`。
> 发布目标：**公共 npm**，包名 `@morehao/ark-iam-react`，scope `@morehao`（免费 public org）。内部四包不对外发布。以下为原始方案留档。

### 1.1 目标

外部业务项目可以像引第三方库一样引入本仓前端能力：

```bash
pnpm add @ark-iam/ui @ark-iam/auth @ark-iam/api @ark-iam/types
```

```tsx
import { ConfigProvider } from 'antd'
import { themeConfig, MainLayout, actionColumn } from '@ark-iam/ui'
import { AuthProvider } from '@ark-iam/auth'
import { createApiClient } from '@ark-iam/api'
```

### 1.2 现状（不可用的事实）

| 包 | `private` | `main` 指向 | build 脚本 | `files` / `publishConfig` |
|---|---|---|---|---|
| `@ark-iam/ui` | `true` | `./src/index.ts` | 无 | 无 |
| `@ark-iam/auth` | `true` | `./src/index.ts` | 无 | 无 |
| `@ark-iam/api` | `true` | `./src/index.ts` | 无 | 无 |
| `@ark-iam/types` | `true` | `./src/index.ts` | 无 | 无 |

- `main` 指向 **TS 源码**，无构建产物；仅靠 `workspace:*` 在 monorepo 内可消费。
- 外部项目：npm 装不到（private）；源码也不可直接装（未编译的 `.ts` 依赖消费者自行转译 + 对齐 tsconfig/依赖）。

### 1.3 前置解耦（**必须先做，否则发出去也没法用**）

| # | 阻塞点 | 现状 | 要改成 |
|---|---|---|---|
| B1 | API baseURL 硬编码 | `frontend/packages/api/src/request.ts`：`axios.create({ baseURL: '/v1' })` + 模块级单例 | 导出 `createApiClient({ baseURL, timeout, headers })`；保留一个默认实例供仓内使用 |
| B2 | OIDC issuer 构建期注入 | `frontend/packages/auth/src/config.ts`：`import.meta.env.VITE_OIDC_ISSUER \|\| '/oidc'` | 改为运行时参数：`AuthProvider` 接收 `issuer / clientId / redirectPath / postLogoutRedirectPath`，`VITE_*` 仅作仓内默认值 |
| B3 | `/config.json` 端点无人消费 | `iam/app_frontend.go` 注册了该端点，前端**未读取**（当前是预留/死代码） | 二选一：<br>(a) 前端启动时先拉 `{base}/config.json` 覆盖默认值（推荐，配合 B2）；<br>(b) 若确认不做运行时配置，则从 `MountFrontend` 删除该端点，避免误导 |

### 1.4 任务清单

| # | 任务 | 要改的文件 | 验收标准 |
|---|---|---|---|
| T1 | 为 4 个包加库构建 | `frontend/packages/{ui,auth,api,types}/package.json` + 新增 `vite.config.lib.ts`（或 `tsup.config.ts`） | `pnpm --filter './packages/*' build` 产出 `dist/`（ESM + `.d.ts`）；`dist/index.js` 可被 node 解析 |
| T2 | 包入口指向产物 | 同上 | `main`/`module`/`types` 指向 `dist`；`exports` 映射 `dist`；`files: ["dist"]` |
| T3 | 去 private + publishConfig | 同上 | `private` 删除或改 `false`；`publishConfig` 指定 registry 与 `access` |
| T4 | 依赖分类：peer vs dep | 4 个包的 `package.json` | `react`、`react-dom`、`antd`、`@ant-design/icons`、`react-router-dom`（ui/auth 用到的）进 `peerDependencies` + `devDependencies`；避免重复打包 |
| T5 | `types` 包只产声明 | `packages/types` | 产出 `.d.ts`（无 JS 或仅空 ESM），消费者类型可用 |
| T6 | 样式/静态资源随包分发 | `packages/ui`（如有 css） | 若存在 CSS，需在 `exports` 暴露样式入口（`./styles.css`），并在 README 说明需手动引入 |
| T7 | 版本与发布通道 | 仓库根（新增 `changesets` 或脚本） | 能独立给 4 个包打版本；发布到私有 registry / GitHub Packages 成功 |
| T8 | 消费者接入文档 | `docs/design/application-integration-guide.md` 新增「前端组件接入」一节；`frontend/packages/ui/README.md`（新建） | 文档给出：安装、`ConfigProvider theme={themeConfig}`、`AuthProvider` 参数、`createApiClient` 用法、最小页面示例 |
| T9 | 冒烟：仓外项目验证 | 新建一个临时仓外 demo（不提交） | 在空目录 `pnpm add` 后，能渲染一个用 `actionColumn` 的表格，且 `AuthProvider` 能完成登录跳转 |

### 1.5 工作量与风险

- 估量：**T1~T9 约 3~5 人日**（含发布通道搭建与仓外冒烟）。
- 风险：
  - **antd 版本耦合**：`ui` 深度依赖 antd 5 的 `ConfigProvider`/`theme`，peer 版本区间过宽会让消费者主题错乱；建议 peer 收紧为 `^5.22`。
  - **设计令牌是代码事实源**（`frontend/packages/ui/src/theme.ts`），发布后即成为对外契约，改动需按 semver。
  - **i18n / 文案**：组件内中文文案硬编码，若未来要国际化，需在发布前先抽出。
  - **构建产物与仓内 dev 的双轨**：`workspace:*` 下是否走 `dist` 还是 `src` 需明确（建议 `publishConfig` + `exports` 条件导出，dev 走 src、发布走 dist），否则开发期要频繁 build。

---

## 2. P1 前端门面包 `@morehao/ark-iam-react`

### 2.1 目标

做到「前端一行接入」：

```tsx
import { IamProvider, IamRoutes } from '@morehao/ark-iam-react'

<IamProvider config={{ issuer, clientId, apiBase }}>
  <IamRoutes basePath="/iam" />
</IamProvider>
```

### 2.2 为什么单列为一项

它**不能**由 P0 的 4 个包直接组合出来——管理台页面目前是 **app 内页面**（`apps/*/src/pages/**`），不是可挂载路由组件。要做门面包必须先把页面拆成受控组件（`IamRoutes` 依赖的是一张路由表，而不是三个独立 Vite 应用）。

### 2.3 任务清单

| # | 任务 | 说明 |
|---|---|---|
| G1 | 抽取页面为可挂载组件 | 把 `apps/platform-admin-web` / `tenant-admin-web` 的 `pages/**` 迁到 `packages/react/src/pages/**`，去掉对 app 级 store/路由的直接依赖，改为 props/context |
| G2 | 路由表与菜单对齐 | `IamRoutes` 暴露的路径需与后端菜单 `code`/`path` 对齐（否则侧边栏点了 404） |
| G3 | 依赖 P0 完成 | 门面包依赖 `@ark-iam/{ui,auth,api,types}` 已可发布 |
| G4 | 与后端 `Domains` 联动 | 后端按域裁剪（如未选 `DomainApplication`）时，前端菜单/路由也要能隐藏对应入口——需一个 `enabledDomains` 配置从 `/config.json` 下发（与 B3 合并解决） |

- 估量：**G1~G4 约 5~8 人日**（页面拆分是主要成本）。
- 依赖：P0 完成。

---

## 3. P2 管理台前端的内嵌/挂载

### 3.1 现状

`iam.MountFrontend` 目前**只内嵌 login-web 的 `dist`**；两个管理台没有内嵌能力。

原因（已核实）：`platform-admin-web` / `tenant-admin-web` 的 `dist/index.html` 使用**绝对路径** `/assets/...`，挂在子路径（如 `/iam/`）会 404，必须按 `base` 重新构建。

### 3.2 两条路

| 路线 | 做法 | 代价 |
|---|---|---|
| P2-A 重建 + embed | 三前端按 `base: '/iam/'` 构建，`MountFrontend` 分别托管 login/console；Go 侧多 embed 两个 dist（体积 +2~3MB） | 中；但不解决「组件复用」，只是整应用内嵌 |
| P2-B 依赖 P1 门面包 | 宿主项目用 `IamRoutes` 自己渲染路由，不需要 embed | 依赖 P1；但这是真正的「组件级」形态 |

> 建议：若确定走 npm 包路线（P0/P1），**P2 的 P2-A 可不做**——宿主用门面包自己渲染即可；只有「完全不想改宿主前端代码」时才需要 P2-A。

---

## 4. P3 后端相关待办

| # | 项 | 说明 | 优先级 |
|---|---|---|---|
| S1 | `/config.json` 端点处置 | 见 B3：要么被前端消费，要么删除 | P0（与 P0 绑定） |
| S2 | IAM 实例的全局收敛 | 当前 `backend/iam` 仍有包级全局：`iam/auth/config.Conf`、`iam/auth/service/svcdb.DB`、`dbclient.SetIamDB/SetRedis` 桥接。**「一进程一 IAM 实例」语义下安全**，但若未来要「一进程多实例」需彻底实例化 | P3（暂不做） |
| S3 | service 构造的 nil 兜底 | 各 `NewXxxSvc` 保留 `if db == nil { db = dbclient.IamDB }`（兼容旧测试）。可在测试全部改为 `testutil.SetupGetter` 后移除 | P3 |
| S4 | `iam` 域依赖表补全 | `domainDeps` 目前只有 `role → menu`；若前端/页面联动发现更多隐式依赖需补 | P2 |
| S5 | 撞名检测的 PG 实测 | `ErrTableConflict` 有 sqlite 用例；需在真实 PostgreSQL 下补一条集成验证 | P2 |

---

## 5. 已完成的记录

| 项 | 结论 | 落点 |
|---|---|---|
| `iam.Config.Redis` 未打通 | 已修：`initRedis` 建连接后同步注册全局 `dbclient.RedisCli`；`Service` 缺省 `iam` | `pkg/dbclient/redis.go`、`iam/app_db.go`、`iam/redis_bridge_test.go` |
| e2e 全流程（真实 PG + Redis） | 8 项全通（建表/install/前端/OIDC discovery/授权码全流程/token 访问/负向 401/落库核对） | `.dsh/docs/notes/2026-11-06-iam-e2e-report.md`（本地，未入库） |
| 可嵌入 IAM 库落地 | R0~R6 完成，`go test ./iam/...` 25 包全绿 | `backend/iam/`、`backend/iam/README.md` |
| P0 前端组件发布为 npm 包 | 已完成（T1~T9，单门面包 `@morehao/ark-iam-react`）：内部四包代码+类型打进 dist，ESM 子路径 `/types /api /auth /ui` 按需引入，对外只发布一个包；peer/dev 分类、脚本发布通道、接入文档、仓外冒烟（actionColumn 表格 + AuthProvider PKCE 登录跳转）+ facade 独立消费（runtime+tsc）均通过。运行期配置优先级 props > `/config.json` > `VITE_*` > 默认 | 见 `docs/design/application-integration-guide.md` §12、`frontend/packages/react/`、`frontend/vite.alias.ts`、`frontend/scripts/release-packages.mjs` |
| 门面包公共 npm 首发 | `@morehao/ark-iam-react@0.1.0` 已发布到公共 npm（scope `@morehao` = 账号用户级 scope，未建 org），`dist-tags.latest=0.1.0`，外部 `pnpm add @morehao/ark-iam-react` 可用。发布口径：起始版本用 `0.1.0`；曾误发 `1.0.0`/`0.0.0-stage` 已 unpublish 清理。发布前需 `npm login` 并把 registry 指到 `https://registry.npmjs.org`（本机默认是 npmmirror 镜像，不可发布） | 公共 npm registry `@morehao/ark-iam-react`；`frontend/packages/react/package.json` |
