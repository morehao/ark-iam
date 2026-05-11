# Cross-Tenant Person Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a person-centered IAM model where one global `person` can join multiple tenants as different `user` records, login requires tenant selection, and switching tenants re-issues tenant-scoped sessions.

**Architecture:** Replace the current user-centered authentication model with a new global `person` domain, keep `user` as the tenant-local member identity, move external identities to `person`, and split auth into two phases: authenticate `person`, then enter a tenant to mint tenant-scoped access and refresh tokens. Preserve the repository's existing Gin + GORM layering and flat `/v1/iam/*` routing style.

**Tech Stack:** Go, Gin, GORM, JWT, MySQL schema SQL, testify, existing DAO/service/controller/router patterns in `backend/apps/iam`

---

## File Structure

### New Files

- `backend/apps/iam/model/person.go`
  Global natural-person model.
- `backend/apps/iam/dao/person.go`
  DAO and query condition for `person`.
- `backend/apps/iam/internal/dto/dtoperson/request.go`
  DTOs for person detail, update, password, and identity binding.
- `backend/apps/iam/internal/dto/dtoperson/response.go`
  DTOs for person detail and tenant list payloads.
- `backend/apps/iam/internal/service/svcperson/person.go`
  Person-facing service methods and person-scoped identity management.
- `backend/apps/iam/internal/controller/ctrperson/person.go`
  Person-facing controller.
- `backend/apps/iam/internal/router/person.go`
  Router registration for person endpoints.
- `backend/apps/iam/internal/service/svcauth/auth_person_flow_test.go`
  Service tests for login, tenant selection, tenant switching, and refresh under the new model.
- `backend/apps/iam/internal/service/svcauth/connector_person_flow_test.go`
  Service tests for connector identity resolution to `person` and tenant selection.
- `backend/apps/iam/internal/service/svcperson/person_identity_test.go`
  Service tests for person identity CRUD.

### Modified Files

- `backend/scripts/sql/iam_schema.sql`
  Replace the old user-centered schema with `person` + updated `user`, `user_identity`, `refresh_token`, and `user_login_log` definitions.
- `backend/apps/iam/model/user.go`
  Remove login-credential fields and add `person_id`, `is_owner`, and `joined_at`.
- `backend/apps/iam/model/user_identity.go`
  Remove `tenant_id` and `user_id`, add `person_id`, `provider`, and `last_used_at`.
- `backend/apps/iam/model/refresh_token.go`
  Add `person_id`, `session_id`, `client_type`, `client_ip`, and `user_agent`.
- `backend/apps/iam/model/user_login_log.go`
  Add `person_id` and `login_type`.
- `backend/apps/iam/dao/user.go`
  Update tenant-user queries to use `person_id` where needed.
- `backend/apps/iam/dao/user_identity.go`
  Rework queries to resolve identities by `person_id` and `issuer + external_subject`.
- `backend/apps/iam/dao/refresh_token.go`
  Rework refresh token lookup and persistence for `person_id + tenant_id + user_id + session_id`.
- `backend/apps/iam/dao/session.go`
  Session listing and revocation under the new token schema.
- `backend/apps/iam/internal/dto/dtoauth/request.go`
  Replace tenant-bound login/register DTOs with person login, select-tenant, switch-tenant, and new refresh semantics.
- `backend/apps/iam/internal/dto/dtoauth/response.go`
  Replace direct token login response with `personToken + tenants` and new userinfo response shape.
- `backend/apps/iam/object/objauth/auth.go`
  Add objects for `PersonTokenInfo`, `TenantOption`, and split `PersonInfo` / `TenantUserInfo` payloads.
- `backend/apps/iam/internal/service/svcauth/auth.go`
  Rewrite auth flow from person authentication to tenant-scoped session minting.
- `backend/apps/iam/internal/service/svcauth/connector_identity.go`
  Resolve external identity to `person`, then to tenant membership.
- `backend/apps/iam/internal/service/svcuser/user_identity.go`
  Remove or redirect user-scoped identity behavior.
- `backend/apps/iam/internal/controller/ctrauth/auth.go`
  Bind new login/selectTenant/switchTenant DTOs.
- `backend/apps/iam/internal/controller/ctrauth/connector.go`
  Return new two-phase auth payload after connector callback.
- `backend/apps/iam/internal/controller/ctrsession/session.go`
  Update session payloads to reflect person-aware sessions.
- `backend/apps/iam/internal/router/auth.go`
  Add `/myTenants`, `/selectTenant`, `/switchTenant`, `/logoutAll` and keep flat paths.
- `backend/apps/iam/internal/router/router.go`
  Register new person router.
- `backend/apps/iam/internal/router/auth_test.go`
  Update route expectations for the new auth endpoints.
- `backend/apps/iam/internal/service/svcsession/session.go`
  Query and revoke person-aware tenant sessions.
- `backend/apps/iam/docs/api_doc.md`
  Update auth model documentation if the repository still uses this manually maintained file.

### Existing Files To Re-Read Before Coding

- `docs/superpowers/specs/2026-05-10-cross-tenant-person-model-design.md`
- `backend/scripts/sql/iam_schema.sql`
- `backend/apps/iam/internal/service/svcauth/auth.go`
- `backend/apps/iam/internal/service/svcauth/connector_identity.go`
- `backend/apps/iam/internal/service/svcsession/session.go`
- `backend/apps/iam/internal/router/auth.go`
- `backend/apps/iam/internal/dto/dtoauth/request.go`
- `backend/apps/iam/internal/dto/dtoauth/response.go`

---

### Task 1: Replace the Core Schema and Models

**Files:**
- Create: `backend/apps/iam/model/person.go`
- Modify: `backend/scripts/sql/iam_schema.sql`
- Modify: `backend/apps/iam/model/user.go`
- Modify: `backend/apps/iam/model/user_identity.go`
- Modify: `backend/apps/iam/model/refresh_token.go`
- Modify: `backend/apps/iam/model/user_login_log.go`
- Create: `backend/apps/iam/dao/person.go`
- Test: `backend/apps/iam/model/connector_model_test.go`

- [ ] **Step 1: Write the failing model and DAO tests**

Add tests that assert the new structs expose the expected field names and table names, and that a `PersonCond` can filter by `username`, `primary_email`, and `primary_phone`.

```go
func TestPersonEntityTableName(t *testing.T) {
	if got := (model.PersonEntity{}).TableName(); got != model.TableNamePerson {
		t.Fatalf("unexpected table name: %s", got)
	}
}

func TestPersonCondBuildConditionSupportsGlobalIdentifiers(t *testing.T) {
	db, err := gdb.NewSQLiteDB()
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonEntity{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	seed := []model.PersonEntity{
		{Username: "alice", PrimaryEmail: "alice@example.com", PrimaryPhone: "13800000001", Name: "Alice"},
		{Username: "bob", PrimaryEmail: "bob@example.com", PrimaryPhone: "13800000002", Name: "Bob"},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	query := db.Model(&model.PersonEntity{})
	cond := &dao.PersonCond{Username: "alice", PrimaryEmail: "alice@example.com"}
	cond.BuildCondition(query, model.TableNamePerson)
	var list []model.PersonEntity
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(list) != 1 || list[0].Username != "alice" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./apps/iam/model ./apps/iam/dao -run 'TestPersonEntityTableName|TestPersonCondBuildConditionSupportsGlobalIdentifiers' -v`
Expected: FAIL because `PersonEntity`, `TableNamePerson`, and `PersonCond` do not exist yet.

- [ ] **Step 3: Write the minimal implementation**

Create `backend/apps/iam/model/person.go` with a `PersonEntity` matching the approved spec. Update `user.go`, `user_identity.go`, `refresh_token.go`, and `user_login_log.go` to remove the old credential semantics and add the new person-centered fields. Create `backend/apps/iam/dao/person.go` with a `PersonCond` and `PersonDao` following the repository's generic DAO pattern.

Use this shape for `PersonEntity`:

```go
type PersonEntity struct {
	gorm.Model
	Username          string          `gorm:"column:username;type:varchar(128);not null;default '';comment:全局用户名"`
	PrimaryEmail      string          `gorm:"column:primary_email;type:varchar(128);not null;default '';comment:主要邮箱"`
	PrimaryPhone      string          `gorm:"column:primary_phone;type:varchar(128);not null;default '';comment:主要手机号"`
	PasswordEncrypted string          `gorm:"column:password_encrypted;type:varchar(256);not null;default '';comment:加密密码"`
	PasswordMethod    string          `gorm:"column:password_method;type:varchar(32);not null;default '';comment:密码加密方式"`
	Name              string          `gorm:"column:name;type:varchar(128);not null;default '';comment:姓名"`
	Avatar            string          `gorm:"column:avatar;type:varchar(2048);not null;default '';comment:头像URL"`
	Profile           json.RawMessage `gorm:"column:profile;type:json;not null;default '{}';comment:配置信息"`
	CustomData        json.RawMessage `gorm:"column:custom_data;type:json;not null;default '{}';comment:自定义数据"`
	IsSuspended       int8            `gorm:"column:is_suspended;type:tinyint(1);not null;default 0;comment:是否挂起"`
	LastSignInAt      *gorm.DeletedAt `gorm:"column:last_sign_in_at;comment:最后登录时间"`
	CreatedBy         uint            `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id"`
	UpdatedBy         uint            `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id"`
	DeletedBy         uint            `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id"`
}
```

Update `iam_schema.sql` so the SQL definitions for `person`, `user`, `user_identity`, `refresh_token`, and `user_login_log` match the spec exactly.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./apps/iam/model ./apps/iam/dao -run 'TestPersonEntityTableName|TestPersonCondBuildConditionSupportsGlobalIdentifiers' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/scripts/sql/iam_schema.sql backend/apps/iam/model/person.go backend/apps/iam/model/user.go backend/apps/iam/model/user_identity.go backend/apps/iam/model/refresh_token.go backend/apps/iam/model/user_login_log.go backend/apps/iam/dao/person.go
git commit -m "feat(iam): add person-centered identity schema"
```

### Task 2: Rewrite Auth DTOs and Person-Centered Login Flow

**Files:**
- Modify: `backend/apps/iam/internal/dto/dtoauth/request.go`
- Modify: `backend/apps/iam/internal/dto/dtoauth/response.go`
- Modify: `backend/apps/iam/object/objauth/auth.go`
- Modify: `backend/apps/iam/internal/service/svcauth/auth.go`
- Modify: `backend/apps/iam/internal/controller/ctrauth/auth.go`
- Modify: `backend/apps/iam/internal/router/auth.go`
- Modify: `backend/apps/iam/internal/router/auth_test.go`
- Test: `backend/apps/iam/internal/service/svcauth/auth_person_flow_test.go`

- [ ] **Step 1: Write the failing auth tests**

Create a service test that verifies:

1. `Login` authenticates a `person` without requiring `tenantID`.
2. `Login` returns a short-lived `personToken` and all joined tenants.
3. `SelectTenant` turns a valid person context into tenant-scoped tokens.
4. `SwitchTenant` rejects tenants not joined by the current `person`.

Use tests like:

```go
func TestLoginReturnsPersonTokenAndTenantOptions(t *testing.T) {
	gCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	repo := &stubAuthPersonRepo{
		personByIdentifier: &model.PersonEntity{Model: gorm.Model{ID: 101}, Username: "alice", PasswordEncrypted: mustHash(t, "Password1"), PasswordMethod: "Argon2id"},
		tenants: []authTenantMembership{{TenantID: 11, UserID: 21, TenantName: "tenant-a"}, {TenantID: 12, UserID: 22, TenantName: "tenant-b"}},
	}
	installAuthPersonStubs(t, repo)
	svc := NewAuthSvc("test-sign-key")
	resp, err := svc.Login(gCtx, &dtoauth.LoginReq{Identifier: "alice", Password: "Password1"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if resp.PersonToken == "" {
		t.Fatal("expected person token")
	}
	if len(resp.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(resp.Tenants))
	}
}

func TestSelectTenantReturnsTenantScopedTokens(t *testing.T) {
	gCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	gincontext.SetUserID(gCtx, 0)
	gincontext.SetValue(gCtx, "personID", uint(101))
	repo := &stubAuthPersonRepo{
		tenants: []authTenantMembership{{TenantID: 11, UserID: 21, TenantName: "tenant-a"}},
	}
	installAuthPersonStubs(t, repo)
	svc := NewAuthSvc("test-sign-key")
	resp, err := svc.SelectTenant(gCtx, &dtoauth.SelectTenantReq{TenantID: 11})
	if err != nil {
		t.Fatalf("SelectTenant returned error: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected tenant-scoped tokens")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./apps/iam/internal/service/svcauth -run 'TestLoginReturnsPersonTokenAndTenantOptions|TestSelectTenantReturnsTenantScopedTokens' -v`
Expected: FAIL because the DTOs and service methods do not match the new auth model.

- [ ] **Step 3: Write minimal implementation**

Change DTOs as follows:

```go
type LoginReq struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type SelectTenantReq struct {
	TenantID uint `json:"tenantID" binding:"required"`
}

type SwitchTenantReq struct {
	TenantID uint `json:"tenantID" binding:"required"`
}

type LoginResp struct {
	PersonToken string                `json:"personToken"`
	Tenants     []objauth.TenantOption `json:"tenants"`
}
```

Add new service methods to `AuthSvc`:

```go
type AuthSvc interface {
	Login(ctx *gin.Context, req *dtoauth.LoginReq) (*dtoauth.LoginResp, error)
	SelectTenant(ctx *gin.Context, req *dtoauth.SelectTenantReq) (*objauth.TokenInfo, error)
	SwitchTenant(ctx *gin.Context, req *dtoauth.SwitchTenantReq) (*objauth.TokenInfo, error)
	MyTenants(ctx *gin.Context) (*dtoauth.MyTenantsResp, error)
	RefreshToken(ctx *gin.Context, req *dtoauth.RefreshTokenReq) (*dtoauth.RefreshTokenResp, error)
	Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error
	LogoutAll(ctx *gin.Context) error
	Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error)
}
```

Implement `Login` to authenticate `person`, mint a short-lived `personToken`, and return the person's tenant memberships. Implement `SelectTenant` and `SwitchTenant` to mint tenant-scoped tokens by loading `person -> tenant -> user`. Update `authRouter` to add:

```go
v1RouterGroup.GET("/myTenants", authCtr.MyTenants)
v1RouterGroup.POST("/selectTenant", authCtr.SelectTenant)
v1RouterGroup.POST("/switchTenant", authCtr.SwitchTenant)
v1RouterGroup.POST("/logoutAll", authCtr.LogoutAll)
```

Update route tests to assert these new endpoints exist.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/router -run 'TestLoginReturnsPersonTokenAndTenantOptions|TestSelectTenantReturnsTenantScopedTokens|TestAuthAndConnectorRoutesUseUnifiedEndpoints' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/apps/iam/internal/dto/dtoauth/request.go backend/apps/iam/internal/dto/dtoauth/response.go backend/apps/iam/object/objauth/auth.go backend/apps/iam/internal/service/svcauth/auth.go backend/apps/iam/internal/controller/ctrauth/auth.go backend/apps/iam/internal/router/auth.go backend/apps/iam/internal/router/auth_test.go backend/apps/iam/internal/service/svcauth/auth_person_flow_test.go
git commit -m "feat(iam): add person-first auth and tenant selection"
```

### Task 3: Move External Identity Resolution to Person

**Files:**
- Modify: `backend/apps/iam/dao/user_identity.go`
- Modify: `backend/apps/iam/internal/service/svcauth/connector_identity.go`
- Modify: `backend/apps/iam/internal/service/svcauth/connector.go`
- Modify: `backend/apps/iam/internal/service/svcuser/user_identity.go`
- Create: `backend/apps/iam/internal/service/svcperson/person.go`
- Create: `backend/apps/iam/internal/service/svcperson/person_identity_test.go`
- Test: `backend/apps/iam/internal/service/svcauth/connector_person_flow_test.go`

- [ ] **Step 1: Write the failing connector and identity tests**

Write tests that verify:

1. Connector callback resolves `issuer + external_subject` to a `person`, not directly to a `user`.
2. If the `person` belongs to multiple tenants, callback returns tenant options instead of minting a tenant token immediately.
3. Person-scoped identity CRUD persists `person_id` and never persists `tenant_id`.

```go
func TestConnectorCallbackReturnsPersonTokenWhenPersonHasMultipleTenants(t *testing.T) {
	gCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	stateStore := &stubConnectorStateStore{state: &ConnectorState{State: "ok", ConnectorID: 11, TenantID: 0, RedirectURI: "https://app.example.com/callback"}}
	resolver := &stubConnectorPersonResolver{
		person: &model.PersonEntity{Model: gorm.Model{ID: 101}, Username: "alice"},
		tenants: []authTenantMembership{
			{TenantID: 11, UserID: 21, TenantName: "tenant-a"},
			{TenantID: 12, UserID: 22, TenantName: "tenant-b"},
		},
	}
	svc := &connectorSvc{
		connectorRepo: &stubConnectorRuntimeRepo{entity: &model.ConnectorEntity{Model: gorm.Model{ID: 11}, Status: connectorStatusEnabled}},
		stateStore:    stateStore,
		personResolver: resolver,
		driverRegistry: stubConnectorDriverRegistryReturningIdentity(StandardIdentity{Issuer: "https://issuer.example.com", Subject: "sub-1", Email: "alice@example.com"}),
		personTokenGenerator: func(_ *gin.Context, person *model.PersonEntity) (string, error) {
			if person.ID != 101 {
				t.Fatalf("unexpected person id: %d", person.ID)
			}
			return "person-token", nil
		},
	}
	resp, err := svc.Callback(gCtx, &dtoconnector.ConnectorCallbackReq{ConnectorID: 11, State: "ok", Code: "code"})
	if err != nil {
		t.Fatalf("Callback returned error: %v", err)
	}
	if resp.PersonToken != "person-token" {
		t.Fatalf("expected person token, got %q", resp.PersonToken)
	}
	if len(resp.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(resp.Tenants))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/service/svcperson -run 'TestConnectorCallbackReturnsPersonTokenWhenPersonHasMultipleTenants|TestPersonIdentityCreatePersistsPersonID' -v`
Expected: FAIL because connector resolution and identity storage are still user-centered.

- [ ] **Step 3: Write minimal implementation**

Update `connector_identity.go` so the resolver flow becomes:

1. Load identity by `issuer + external_subject`.
2. Resolve or create a `person`.
3. Load tenant memberships for that `person`.
4. Return a person-scoped auth payload.

Use interfaces like:

```go
type connectorPersonRepository interface {
	GetByID(ctx context.Context, id uint) (*model.PersonEntity, error)
	Insert(ctx context.Context, person *model.PersonEntity) error
}

type connectorPersonIdentityRepository interface {
	GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error)
	Insert(ctx context.Context, entity *model.UserIdentityEntity) error
	UpdateBinding(ctx context.Context, identityID, personID uint, issuer string, detail []byte) error
}
```

Move manual identity CRUD from `svcuser` to `svcperson`, keeping `svcuser` free of global identity semantics.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/service/svcperson -run 'TestConnectorCallbackReturnsPersonTokenWhenPersonHasMultipleTenants|TestPersonIdentityCreatePersistsPersonID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/apps/iam/dao/user_identity.go backend/apps/iam/internal/service/svcauth/connector_identity.go backend/apps/iam/internal/service/svcauth/connector.go backend/apps/iam/internal/service/svcuser/user_identity.go backend/apps/iam/internal/service/svcperson/person.go backend/apps/iam/internal/service/svcperson/person_identity_test.go backend/apps/iam/internal/service/svcauth/connector_person_flow_test.go
git commit -m "feat(iam): bind external identities to person"
```

### Task 4: Update Session, Userinfo, and Person/User Boundaries

**Files:**
- Modify: `backend/apps/iam/dao/session.go`
- Modify: `backend/apps/iam/internal/service/svcsession/session.go`
- Modify: `backend/apps/iam/internal/controller/ctrsession/session.go`
- Modify: `backend/apps/iam/internal/service/svcuser/user.go`
- Create: `backend/apps/iam/internal/controller/ctrperson/person.go`
- Create: `backend/apps/iam/internal/router/person.go`
- Modify: `backend/apps/iam/internal/router/router.go`
- Modify: `backend/apps/iam/docs/api_doc.md`

- [ ] **Step 1: Write the failing session and userinfo tests**

Write tests that verify:

1. Session listing returns person-aware tenant sessions.
2. `userinfo` returns both `personInfo` and `userInfo`.
3. `svcuser` no longer assumes user is the global login principal.

```go
func TestUserinfoReturnsPersonAndTenantUser(t *testing.T) {
	gCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	gincontext.SetUserID(gCtx, 21)
	gincontext.SetTenantID(gCtx, 11)
	gincontext.SetValue(gCtx, "personID", uint(101))
	svc := &authSvc{jwtSecret: "test-sign-key"}
	resp, err := svc.Userinfo(gCtx, &dtoauth.UserinfoReq{})
	if err != nil {
		t.Fatalf("Userinfo returned error: %v", err)
	}
	if resp.PersonInfo.PersonID != 101 || resp.UserInfo.UserID != 21 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/service/svcsession -run 'TestUserinfoReturnsPersonAndTenantUser|TestSessionListReturnsPersonAwareTenantSessions' -v`
Expected: FAIL because userinfo and session services still only know the old `user` model.

- [ ] **Step 3: Write minimal implementation**

Update session DAO and service to load and revoke sessions by `person_id`, `tenant_id`, `user_id`, and `session_id`. Update `UserinfoResp` to include split payloads:

```go
type UserinfoResp struct {
	PersonInfo objauth.PersonInfo     `json:"personInfo"`
	UserInfo   objauth.TenantUserInfo `json:"userInfo"`
}
```

Add `ctrperson` and `personRouter` for future person profile endpoints, even if the initial controller only exposes `GET /person/detail` and `POST /person/updatePassword`. Keep `svcuser` focused on tenant-local user data.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/service/svcsession ./apps/iam/internal/router -run 'TestUserinfoReturnsPersonAndTenantUser|TestSessionListReturnsPersonAwareTenantSessions|TestAuthAndConnectorRoutesUseUnifiedEndpoints' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/apps/iam/dao/session.go backend/apps/iam/internal/service/svcsession/session.go backend/apps/iam/internal/controller/ctrsession/session.go backend/apps/iam/internal/service/svcuser/user.go backend/apps/iam/internal/controller/ctrperson/person.go backend/apps/iam/internal/router/person.go backend/apps/iam/internal/router/router.go backend/apps/iam/docs/api_doc.md
git commit -m "feat(iam): separate person identity from tenant user session"
```

### Task 5: Full Verification

**Files:**
- Modify: `docs/superpowers/specs/2026-05-10-cross-tenant-person-model-design.md`
- Modify: `docs/superpowers/plans/2026-05-10-cross-tenant-person-model-plan.md`

- [x] **Step 1: Run focused auth and identity tests**

Run: `go test ./apps/iam/internal/service/svcauth ./apps/iam/internal/service/svcperson ./apps/iam/internal/service/svcsession -v`
Expected: PASS.
Result: PASS.

- [x] **Step 2: Run router and DAO tests**

Run: `go test ./apps/iam/internal/router ./apps/iam/dao ./apps/iam/model -v`
Expected: PASS.
Result: PASS.

- [x] **Step 3: Run the application test suite**

Run: `make test APP=iam`
Expected: PASS.
Result: PASS.

- [x] **Step 4: Update the plan and spec status if needed**

Fixed `TestUserIdentityPageListUsesContextTenant` to assert `PersonID` instead of removed `TenantID`.
Fixed `TestRegisterAllowsEmailOnlyIdentifier` and `TestRegisterAllowsPhoneOnlyIdentifier` to also swap `authPersonStore`.

- [x] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(iam): implement cross-tenant person model"
```

---

## Self-Review

### Spec Coverage

This plan covers the approved spec sections as follows:

1. `person` global identity model: Task 1.
2. `user` as tenant-local member identity: Tasks 1 and 4.
3. `user_identity` bound to `person`: Task 3.
4. Login -> tenant selection -> tenant-scoped session flow: Task 2.
5. Tenant switching and person-aware refresh sessions: Tasks 2 and 4.
6. Codebase directory placement and module boundaries: Tasks 3 and 4.

No spec requirement is left without at least one task.

### Placeholder Scan

Checked this plan for `TBD`, `TODO`, `implement later`, ambiguous “add validation” language, and task references without concrete files or commands. None remain.

### Type Consistency

The plan consistently uses:

1. `PersonEntity`
2. `person_id`
3. `SelectTenantReq`
4. `SwitchTenantReq`
5. `PersonToken`
6. `TenantOption`
7. `TenantUserInfo`

No later task introduces a conflicting type name.

---

## Implementation Status

- [x] Task 1: Replace the Core Schema and Models — DONE
- [x] Task 2: Rewrite Auth DTOs and Person-Centered Login Flow — DONE
- [x] Task 3: Move External Identity Resolution to Person — DONE
- [x] Task 4: Update Session, Userinfo, and Person/User Boundaries — DONE
- [x] Task 5: Full Verification — DONE

### Self-Check Conclusion

`make test APP=iam` 全量测试全部通过。

修复内容：
1. `TestUserIdentityPageListUsesContextTenant` — 将旧 `TenantID` 字段断言改为 person-centered 语义下的 `PersonID` 断言
2. `TestRegisterAllowsEmailOnlyIdentifier` / `TestRegisterAllowsPhoneOnlyIdentifier` — 补充 `authPersonStore` 的 stub swap，避免 nil pointer dereference
