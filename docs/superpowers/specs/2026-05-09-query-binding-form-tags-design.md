# IAM Query 绑定 Form 标签修复设计

## 文档信息

- **创建时间**：2026-05-09
- **版本**：v1.0
- **状态**：待审阅

---

## 一、问题背景

当前 `backend/apps/iam` 中存在一批控制器使用 `ctx.ShouldBindQuery(&req)` 处理查询参数，但对应 DTO 仅声明了 `json` 标签，未统一声明 `form` 标签。

这会带来两个直接问题：

1. 查询参数名与 DTO 字段绑定行为不稳定，尤其是 lowerCamel 风格参数如 `tenantID`、`userID`、`connectorId`。
2. 带 `binding:"required"` 的详情类查询请求在 query 场景下可能拿到零值，导致错误的 400、误查或空查询。

这类问题属于接口可用性和审计稳定性问题，修复应优先保证最小改动，不扩散到 service/dao 语义。

---

## 二、目标

本次修复目标仅限于：

1. 为所有实际通过 `ShouldBindQuery` 绑定的 DTO 字段补齐 `form` 标签。
2. 保持已有 `json` 标签不变，避免影响 JSON body 接口。
3. 为关键 query 绑定路径补充回归测试，验证 query 参数名能够稳定映射到 DTO 字段。

本次不包含：

- service/dao 行为调整
- 路由路径或 HTTP 方法调整
- Swagger 批量重构
- 非 query 场景 DTO 清理

---

## 三、设计方案

### 方案 A：仅为 Query DTO 补 `form` 标签（推荐）

做法：

1. 识别所有 `ShouldBindQuery` 控制器入口。
2. 仅修改这些入口实际使用的 DTO 字段，为其增加与现有 `json` 一致的 `form` 标签。
3. 保留原有字段名和结构体定义，不拆 DTO。

优点：

- 改动最小
- 与 Gin 绑定机制一致
- 风险低，便于回归

缺点：

- 需要逐个 DTO 检查，避免遗漏

### 方案 B：controller 改为手动读取 query 参数

做法：

1. controller 不再使用 `ShouldBindQuery`
2. 改成 `ctx.Query` / `ctx.DefaultQuery` / `strconv` 手动解析

优点：

- 行为显式

缺点：

- 分散重复逻辑多
- 会绕开现有 binding/validation 习惯
- 不适合本轮最小修复

### 方案 C：单独拆分 Query DTO

做法：

1. 为 query 场景单独定义新 DTO
2. body/query 各自独立维护标签

优点：

- 语义更清晰

缺点：

- 改动面大
- 当前问题不需要这么重的重构

### 结论

采用方案 A。

---

## 四、修改边界

优先覆盖以下 DTO 文件中被 `ShouldBindQuery` 实际使用的结构体字段：

- `backend/apps/iam/internal/dto/dtouser/request.go`
- `backend/apps/iam/internal/dto/dtotenant/request.go`
- `backend/apps/iam/internal/dto/dtotenant/system_request.go`
- `backend/apps/iam/internal/dto/dtoauth/request.go`
- `backend/apps/iam/internal/dto/dtopermission/request.go`
- 如实际存在 query 绑定但未纳入上面列表的 DTO，同样补齐

控制器文件仅用于确认入口，不计划修改绑定方式：

- `backend/apps/iam/internal/controller/**`

---

## 五、测试策略

采用 TDD：先补失败测试，再补 `form` 标签。

### 5.1 测试目标

至少覆盖三类典型场景：

1. 详情查询：`GET` + 必填 ID
2. 按用户查询：`GET` + 业务 ID
3. 列表查询：`GET` + 多个筛选字段

### 5.2 推荐首批回归点

优先选择已有风险较高、调用频繁的接口：

1. `user/detail` 对应 `dtouser.UserDetailReq`
2. `user/getUserLoginLogByUser` 或 `user/getUserDepartmentRelationByUser`
3. `tenant/detail` 或 `department/tree`
4. `connector/detail` 或 `connector/callback` 相关 query DTO（若当前确实通过 `ShouldBindQuery`）

### 5.3 验证标准

修复后应满足：

1. query 参数按文档字段名传入时，可正确绑定到 DTO。
2. `binding:"required"` 在 query 场景下稳定生效。
3. 不影响现有 JSON body 接口。

---

## 六、风险与约束

### 风险

1. 某些 DTO 同时用于 body 和 query，新增 `form` 标签时如果字段名写错，会造成 query 绑定偏差。
2. 少数接口可能依赖默认字段名推断，补标签后会把旧的非标准参数名固定为标准 lowerCamel 名。

### 约束

1. 只做标签级修复，不顺手重构 controller/service。
2. 只在有 query 绑定事实的 DTO 上补 `form`，不做全量无差别添加。
3. 每补一批都要有对应测试或验证命令覆盖。

---

## 七、预期结果

完成后，IAM 中使用 `ShouldBindQuery` 的主要接口将能稳定接收 lowerCamel query 参数，避免因缺少 `form` 标签导致的绑定失效和必填校验异常。
