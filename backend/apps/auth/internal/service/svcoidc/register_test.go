package svcoidc

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/testutil"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type appSeedApp struct {
	clientCode string
	allow      *bool
}

func newSeedDB(t *testing.T, apps []appSeedApp) *gorm.DB {
	t.Helper()
	// 迁移全表：person/tenant/user/department 供 createTenant 落库；app/client 供策略判定。
	db := testutil.SetupSQLite(t,
		&model.PersonEntity{}, &model.TenantEntity{}, &model.UserEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{},
		&model.ApplicationClientEntity{}, &model.ApplicationEntity{},
		// 建租户链路还会写入租户自服务的角色/授权/订阅（权限开通）
		&model.MenuEntity{}, &model.RoleEntity{}, &model.RoleMenuEntity{},
		&model.TenantApplicationEntity{}, &model.UserRoleEntity{},
	)
	// 权限开通依赖租户自服务应用（真实环境由 pkg/seed 写入）
	if err := db.Create(&model.ApplicationEntity{Code: tenant.ProvisionAppCode, Name: "租户自服务", Status: model.AppStatusEnable}).Error; err != nil {
		t.Fatalf("seed tenant_admin application: %v", err)
	}
	for _, a := range apps {
		appEntity := &model.ApplicationEntity{Code: "app-" + a.clientCode, AllowPersonCreateTenant: a.allow}
		if err := db.Create(appEntity).Error; err != nil {
			t.Fatalf("seed app %s: %v", a.clientCode, err)
		}
		client := &model.ApplicationClientEntity{Code: a.clientCode, AppID: appEntity.ID}
		client.ID = client.AppID
		client.RedirectURIs = datatypes.JSON(`[]`)
		client.PostLogoutRedirectURIs = datatypes.JSON(`[]`)
		client.GrantTypes = datatypes.JSON(`["authorization_code"]`)
		client.ResponseTypes = datatypes.JSON(`["code"]`)
		client.AllowedOrigins = datatypes.JSON(`[]`)
		client.DefaultScopes = datatypes.JSON(`["openid"]`)
		if err := db.Create(client).Error; err != nil {
			t.Fatalf("seed client %s: %v", a.clientCode, err)
		}
	}
	return db
}

func newAuthReq(t *testing.T, provider *OIDCProvider, clientID string) op.AuthRequest {
	t.Helper()
	req, err := provider.Storage.CreateAuthRequest(t.Context(), &oidc.AuthRequest{
		ClientID:     clientID,
		RedirectURI:  "https://client.example.com/callback",
		State:        "s-1",
		Scopes:       []string{oidc.ScopeOpenID},
		ResponseType: oidc.ResponseTypeCode,
		ResponseMode: oidc.ResponseModeQuery,
	}, "")
	if err != nil {
		t.Fatalf("CreateAuthRequest: %v", err)
	}
	return req
}

func registerSvc(provider *OIDCProvider, db *gorm.DB, tenants func(*gin.Context, string) ([]objauth.TenantOption, error)) *oidcAuthSvc {
	return &oidcAuthSvc{
		provider:             provider,
		authSvc:              &fakePasswordAuthenticator{tenantsForPerson: tenants},
		applicationClientDao: func() *dao.ApplicationClientDao { return dao.NewApplicationClientDao(dao.WithDBGetter(dbGetter(db))) },
		applicationDao:       func() *dao.ApplicationDao { return dao.NewApplicationDao(dao.WithDBGetter(dbGetter(db))) },
	}
}

func TestRegisterPersonDisallowedWhenAppPolicyFalse(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true}}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider: %v", err)
	}
	db := newSeedDB(t, []appSeedApp{{clientCode: "cid-1", allow: model.BoolPtr(false)}})
	authReq := newAuthReq(t, provider, "cid-1")

	svc := registerSvc(provider, db, nil)
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/registerPerson", nil)
	if _, err := svc.RegisterPerson(ginCtx, &dtooidc.RegisterPersonReq{AuthRequestID: authReq.GetID(), Username: "alice", Password: "Password123"}); err == nil {
		t.Fatal("expected register to fail when app disallows")
	}
}

func TestRegisterPersonCreatesAndBindsPerson(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true}}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider: %v", err)
	}
	db := newSeedDB(t, []appSeedApp{{clientCode: "cid-1", allow: model.BoolPtr(true)}})
	authReq := newAuthReq(t, provider, "cid-1")

	svc := registerSvc(provider, db, func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
		return nil, nil // 零租户
	})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/registerPerson", nil)
	resp, err := svc.RegisterPerson(ginCtx, &dtooidc.RegisterPersonReq{AuthRequestID: authReq.GetID(), Username: "alice", Password: "Password123", Name: "Alice"})
	if err != nil {
		t.Fatalf("RegisterPerson: %v", err)
	}
	if resp.PersonID == "" {
		t.Fatal("expected non-empty personID")
	}
	if !resp.AllowPersonCreateTenant {
		t.Fatal("expected allowPersonCreateTenant=true for zero-tenant person in allowing app")
	}
	p, perr := dao.NewPersonDao().GetByCond(t.Context(), &dao.PersonCond{Username: "alice"})
	if perr != nil || p == nil || p.ID == "" {
		t.Fatalf("expected person persisted, err:%v", perr)
	}
	upd, uerr := provider.Storage.AuthRequestByID(t.Context(), authReq.GetID())
	if uerr != nil {
		t.Fatalf("AuthRequestByID: %v", uerr)
	}
	if upd.Done() {
		t.Fatal("expected auth request NOT done after registerPerson")
	}
	if upd.GetSubject() != oidcop.BuildSubject(resp.PersonID) {
		t.Fatalf("expected subject %s, got %s", oidcop.BuildSubject(resp.PersonID), upd.GetSubject())
	}
}

func TestRegisterPersonExistingPersonRequiresPasswordLogin(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true}}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider: %v", err)
	}
	db := newSeedDB(t, []appSeedApp{{clientCode: "cid-1", allow: model.BoolPtr(true)}})
	existing := &model.PersonEntity{
		Username:          model.StrPtr("alice"),
		PasswordEncrypted: "keep-me",
		PasswordMethod:    model.PasswordMethodBcrypt,
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("seed existing person: %v", err)
	}
	authReq := newAuthReq(t, provider, "cid-1")

	svc := registerSvc(provider, db, func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
		return nil, nil
	})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/registerPerson", nil)
	resp, err := svc.RegisterPerson(ginCtx, &dtooidc.RegisterPersonReq{AuthRequestID: authReq.GetID(), Username: "alice", Password: "NewPass123"})
	if err != nil {
		t.Fatalf("RegisterPerson: %v", err)
	}
	if !resp.RequiresPasswordLogin {
		t.Fatal("expected RequiresPasswordLogin=true for existing person")
	}
	if resp.PersonID != existing.ID {
		t.Fatalf("expected existing person %s, got %s", existing.ID, resp.PersonID)
	}
	after, aerr := dao.NewPersonDao().GetByCond(t.Context(), &dao.PersonCond{Username: "alice"})
	if aerr != nil || after == nil {
		t.Fatalf("get person after: %v", aerr)
	}
	if after.PasswordEncrypted != "keep-me" {
		t.Fatalf("expected password NOT overwritten, got %q", after.PasswordEncrypted)
	}
	upd, uerr := provider.Storage.AuthRequestByID(t.Context(), authReq.GetID())
	if uerr != nil {
		t.Fatalf("AuthRequestByID: %v", uerr)
	}
	if upd.Done() {
		t.Fatal("expected auth request NOT done for existing person")
	}
	if upd.GetSubject() == oidcop.BuildSubject(existing.ID) {
		t.Fatal("expected auth request NOT bound to existing person")
	}
}

func TestCreateTenantSucceedsForZeroTenantPerson(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true}}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider: %v", err)
	}
	db := newSeedDB(t, []appSeedApp{{clientCode: "cid-1", allow: model.BoolPtr(true)}})
	p := &model.PersonEntity{Username: model.StrPtr("bob"), Name: "Bob", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}
	authReq := newAuthReq(t, provider, "cid-1")
	if err := provider.Storage.CompleteAuthRequest(t.Context(), authReq.GetID(), oidcop.BuildSubject(p.ID), time.Now(), []string{"pwd"}, "", "", false); err != nil {
		t.Fatalf("bind person: %v", err)
	}

	svc := registerSvc(provider, db, func(ctx *gin.Context, personID string) ([]objauth.TenantOption, error) {
		return nil, nil // 零租户
	})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/createTenant", nil)
	res, err := svc.CreateTenant(ginCtx, &dtooidc.CreateTenantReq{AuthRequestID: authReq.GetID(), TenantName: "Acme", TenantCode: "acme"})
	if err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}
	if res.TenantID == "" || res.PersonID != p.ID {
		t.Fatalf("unexpected resp: %#v", res)
	}
	tenants, tErr := dao.NewTenantDao().GetListByCond(t.Context(), &dao.TenantCond{})
	if tErr != nil || len(tenants) == 0 {
		t.Fatalf("expected tenant persisted, err:%v", tErr)
	}
	users, uErr := dao.NewUserDao().GetListByCond(t.Context(), &dao.UserCond{PersonID: p.ID, TenantID: res.TenantID})
	if uErr != nil || len(users) == 0 || !users[0].IsOwner {
		t.Fatalf("expected owner user, got users:%#v err:%v", users, uErr)
	}
	// owner 走与平台侧同一份开通实现：builtin 来源 + 归属根部门 + 内置租户管理员角色
	if users[0].Source != model.UserSourceBuiltin {
		t.Errorf("owner source = %q, want %q (平台重置内置管理员密码依赖该标记)", users[0].Source, model.UserSourceBuiltin)
	}
	rootDept, oErr := dao.NewDepartmentDao().GetByCond(t.Context(), &dao.DepartmentCond{TenantID: res.TenantID})
	if oErr != nil || rootDept == nil {
		t.Fatalf("expected root department, err:%v dept:%#v", oErr, rootDept)
	}
	deptUsers, ouErr := dao.NewDepartmentUserDao().GetListByCond(t.Context(), &dao.DepartmentUserCond{
		TenantID: res.TenantID, UserID: users[0].ID, RelationType: model.DeptUserRelationPrimary,
	})
	if ouErr != nil || len(deptUsers) != 1 || deptUsers[0].DepartmentID != rootDept.ID {
		t.Fatalf("expected owner attached to root dept, got %#v err:%v", deptUsers, ouErr)
	}
	role, rErr := dao.NewRoleDao().GetByCond(t.Context(), &dao.RoleCond{
		TenantID:  res.TenantID,
		Source:    model.RoleSourceBuiltin,
		AdminType: model.SysAdminTypeAdmin,
	})
	if rErr != nil || role == nil {
		t.Fatalf("expected provisioned tenant admin role, err:%v role:%#v", rErr, role)
	}
	userRoles, urErr := dao.NewUserRoleDao().GetListByCond(t.Context(), &dao.UserRoleCond{
		TenantID: res.TenantID, UserID: users[0].ID, RoleID: role.ID,
	})
	if urErr != nil || len(userRoles) != 1 {
		t.Fatalf("expected owner granted tenant admin role, got %#v err:%v", userRoles, urErr)
	}
}

func TestLoginConfig(t *testing.T) {
	newProvider := func(t *testing.T) *OIDCProvider {
		testsetup.Initialize(testsetup.AppNameAuth)
		t.Cleanup(func() { testsetup.Done(testsetup.AppNameAuth) })
		appconfig.Conf = &pkgconfig.Config{OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true}}
		provider, err := SetupOIDCProvider()
		if err != nil {
			t.Fatalf("SetupOIDCProvider: %v", err)
		}
		return provider
	}

	t.Run("allow register => true", func(t *testing.T) {
		provider := newProvider(t)
		db := newSeedDB(t, []appSeedApp{{clientCode: "cfg-1", allow: model.BoolPtr(true)}})
		authReq := newAuthReq(t, provider, "cfg-1")
		svc := registerSvc(provider, db, nil)
		ginCtx, _ := gin.CreateTestContext(nil)
		ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login-config", nil)
		res, err := svc.LoginConfig(ginCtx, authReq.GetID())
		if err != nil {
			t.Fatalf("LoginConfig: %v", err)
		}
		if !res.AllowPersonCreateTenant {
			t.Fatal("expected allowPersonCreateTenant=true")
		}
	})

	t.Run("disallow register => false", func(t *testing.T) {
		provider := newProvider(t)
		db := newSeedDB(t, []appSeedApp{{clientCode: "cfg-2", allow: model.BoolPtr(false)}})
		authReq := newAuthReq(t, provider, "cfg-2")
		svc := registerSvc(provider, db, nil)
		ginCtx, _ := gin.CreateTestContext(nil)
		ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login-config", nil)
		res, err := svc.LoginConfig(ginCtx, authReq.GetID())
		if err != nil {
			t.Fatalf("LoginConfig: %v", err)
		}
		if res.AllowPersonCreateTenant {
			t.Fatal("expected allowPersonCreateTenant=false")
		}
	})

	t.Run("invalid authRequest => error", func(t *testing.T) {
		provider := newProvider(t)
		db := newSeedDB(t, nil)
		svc := registerSvc(provider, db, nil)
		ginCtx, _ := gin.CreateTestContext(nil)
		ginCtx.Request, _ = http.NewRequest(http.MethodPost, "/oidc/login-config", nil)
		if _, err := svc.LoginConfig(ginCtx, "missing-id"); err == nil {
			t.Fatal("expected error for invalid authRequestID")
		}
	})
}
