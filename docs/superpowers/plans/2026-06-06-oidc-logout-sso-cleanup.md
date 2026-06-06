# OIDC 登出 SSO Session 清理 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除中间件 hack，通过 zitadel/oidc 标准扩展点 `TerminateSession` 清理 Redis SSO session，在 `SSOLogin` handler 中添加防御性 cookie 清除。

**Architecture:** 三层防御 — `TerminateSession`（主路径，标准扩展点）清除 Redis session → `SSOLogin`（防御路径）验证失败时清除孤儿 cookie → `/logged-out`（兜底路径）已存在清除 cookie 逻辑。

**Tech Stack:** Go, Gin, zitadel/oidc v3, Redis (go-redis)

---

### Task 1: 移除路由层中间件 hack

**Files:**
- Modify: `backend/apps/iam/internal/router/oidc.go`

- [ ] **Step 1: 删除中间件函数和辅助函数**

删除 `parseJWTSub`（第 22-38 行）、`parsePersonIDFromSubject`（第 40-54 行）、`endSessionCleanup`（第 56-72 行）。

具体操作：用 Edit 工具删除第 22-72 行。

- [ ] **Step 2: 删除 import**

删除 `"encoding/base64"`、`"encoding/json"`、`"strings"` import 行（第 6-7 行、第 9 行）。

文件 import 恢复到：

```go
import (
	"context"
	"crypto/rsa"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gmiddleware/ginmiddleware"
	"github.com/morehao/golib/biz/gserver/ginserver"
)
```

- [ ] **Step 3: 删除中间件注册行**

删除 `InitOIDC` 中的 `oidcGroup.Use(endSessionCleanup)`（第 99 行）。

- [ ] **Step 4: 验证编译**

```bash
make build APP=iam
```
Expected: 编译通过

- [ ] **Step 5: 提交**

```bash
git add backend/apps/iam/internal/router/oidc.go
git commit -m "refactor: remove end_session middleware hack from oidc router"
```

---

### Task 2: SSOLogin 防御 — 验证失败时清除 cookie

**Files:**
- Modify: `backend/apps/iam/internal/controller/ctroidc/oidc.go:58-62`

- [ ] **Step 1: 在 SSOLogin 失败分支添加 cookie 清除**

修改 `SSOLogin` 方法，在 `CompleteLoginBySession` 失败后（第 59 行）添加 `SetCookie`：

```go
	continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx.Request.Context(), authRequestID, sessionID)
	if err != nil {
		ctx.SetCookie("iam_sso_session", "", -1, "/", "", false, true)
		frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
		ctx.Redirect(302, frontendURL)
		return
	}
```

- [ ] **Step 2: 验证编译**

```bash
make build APP=iam
```
Expected: 编译通过

- [ ] **Step 3: 运行测试**

```bash
go test ./apps/iam/internal/service/svcoidc/... -v -count=1 -run "TestCompleteLogin"
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add backend/apps/iam/internal/controller/ctroidc/oidc.go
git commit -m "fix: clear iam_sso_session cookie on SSO login validation failure"
```

---

### Task 3: TerminateSession 添加日志

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/persistent_store.go:256-262`

- [ ] **Step 1: 添加 slog import**

在 `persistent_store.go` 的 import 中添加 `"log/slog"`。

- [ ] **Step 2: 在 TerminateSession 中添加日志**

```go
func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
	personID, err := parseOIDCSubject(userID)
	if err != nil {
		slog.WarnContext(ctx, "TerminateSession: failed to parse userID", "userID", userID, "error", err)
		return nil
	}
	slog.InfoContext(ctx, "TerminateSession: revoking SSO sessions", "userID", userID, "personID", personID, "clientID", clientID)
	return NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID)
}
```

- [ ] **Step 3: 验证编译**

```bash
make build APP=iam
```
Expected: 编译通过

- [ ] **Step 4: 运行测试**

```bash
go test ./apps/iam/internal/service/svcoidc/... -v -count=1
```
Expected: PASS（除已有的 `TestOIDCClientLoginURLUsesConfiguredFrontend` 外）

- [ ] **Step 5: 提交**

```bash
git add backend/apps/iam/internal/service/svcoidc/persistent_store.go
git commit -m "feat: add logging to TerminateSession for SSO session cleanup"
```

---

### Task 4: 验证与收尾

- [ ] **Step 1: 运行完整测试**

```bash
make test APP=iam
```
Expected: 除已有无关失败外全部 PASS

- [ ] **Step 2: 查看最终 diff**

```bash
git diff origin/feature/e2e --stat
```

- [ ] **Step 3: 手动验证清单**

以下需启动服务后手动验证：
1. 登录 → 进入首页
2. 点击登出 → 跳转到登录页
3. 检查后端日志：应出现 `TerminateSession: revoking SSO sessions` 日志
4. 点击登录 → 出现登录表单，需要输入账号密码（不再自动登录）
