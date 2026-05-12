# WithIamContext 重构设计

## 背景

`pkg/testsetup/init.go` 中的 `WithUserContext` 直接 hardcode 了查询 `iam` 表的逻辑，导致 `pkg` 包依赖 `iam` 应用。

## 目标

- `pkg` 包不依赖任何业务应用
- 由调用者（`iam` 应用）提供获取用户上下文的函数

## 设计

### 类型定义

```go
type IamContextProvider func(ctx context.Context) (userID, tenantID, personID, deptID uint, err error)
```

### 函数签名

```go
func WithIamContext(provider IamContextProvider) testkit.Option
```

### 行为

- `WithIamContext` 接受一个 `IamContextProvider` 函数
- 调用者提供 provider 函数，内部通过 `iam/dao` 层获取用户信息
- 如果 provider 未注入或返回 error，直接 panic

### 用法示例

```go
// iam 应用测试代码
testsetup.NewContext(
    testsetup.WithIamContext(func(ctx context.Context) (userID, tenantID, personID, deptID uint, err error) {
        // 使用 iam/dao 获取用户信息
        return
    }),
)
```

## 文件变更

| 文件 | 操作 |
|------|------|
| `pkg/testsetup/init.go` | 重命名 `WithUserContext` → `WithIamContext`，修改签名 |