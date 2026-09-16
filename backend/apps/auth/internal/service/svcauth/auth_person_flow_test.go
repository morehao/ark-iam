package svcauth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakeAuthPersonStore struct {
	getByIDFunc   func(ctx context.Context, id string) (*model.PersonEntity, error)
	getByCondFunc func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error)
	insertFunc    func(ctx context.Context, entity *model.PersonEntity) error
}

func (f *fakeAuthPersonStore) GetByID(ctx context.Context, id string) (*model.PersonEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthPersonStore) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.PersonEntity, error) {
	if f.getByCondFunc == nil {
		return nil, nil
	}
	personCond, _ := cond.(*dao.PersonCond)
	return f.getByCondFunc(ctx, personCond)
}

func (f *fakeAuthPersonStore) Insert(ctx context.Context, entity *model.PersonEntity) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, entity)
}

type fakeAuthTenantStore struct {
	getByIDFunc       func(ctx context.Context, id string) (*model.TenantEntity, error)
	getPageListByCond func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, int64, error)
	getListByCondFunc func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error)
}

func (f *fakeAuthTenantStore) GetByID(ctx context.Context, id string) (*model.TenantEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthTenantStore) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.TenantEntityList, int64, error) {
	if f.getPageListByCond == nil {
		return nil, 0, nil
	}
	tenantCond, _ := cond.(*dao.TenantCond)
	return f.getPageListByCond(ctx, tenantCond)
}

func (f *fakeAuthTenantStore) GetListByCond(ctx context.Context, cond gormdao.Cond) (model.TenantEntityList, error) {
	if f.getListByCondFunc == nil {
		return nil, nil
	}
	tenantCond, _ := cond.(*dao.TenantCond)
	return f.getListByCondFunc(ctx, tenantCond)
}

func TestMyTenantsReturnsCurrentPersonTenantList(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	var userLookup *dao.UserCond
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				userLookup = cond
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, TenantID: "11", PersonID: "88", Name: "tenant-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				userLookup = cond
				return model.UserEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, TenantID: "11", PersonID: "88", Name: "tenant-user-a"},
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "102"}}, TenantID: "12", PersonID: "88", Name: "tenant-user-b"},
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
				return model.TenantEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "11"}}, Name: "租户A", Status: model.TenantStatusActive, Tag: "a"},
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "12"}}, Name: "租户B", Status: model.TenantStatusActive, Tag: "b"},
				}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	resp, err := svc.MyTenants(ginCtx, &dtoauth.MyTenantsReq{})
	if err != nil {
		t.Fatalf("MyTenants returned error: %v", err)
	}
	if userLookup == nil || userLookup.PersonID != "88" {
		t.Fatalf("expected tenant lookup to use personID 88, got %+v", userLookup)
	}
	if resp == nil || len(resp.List) != 2 {
		t.Fatalf("expected two tenants, got %#v", resp)
	}
	if resp.List[0].TenantID != "11" || resp.List[1].TenantID != "12" {
		t.Fatalf("expected joined tenant IDs [11 12], got %#v", resp.List)
	}
}

// TestMyTenantsRejectsWhenAllTenantsSuspended 挂起租户既不进选择列表也不能作为默认租户；
// 成员关系全部指向挂起租户时明确拒绝（100207），不与"零租户可自助建租户"路径混同。
func TestMyTenantsRejectsWhenAllTenantsSuspended(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, TenantID: "11", PersonID: "88", Name: "tenant-user-a"},
				}, nil
			},
		}
	})
	defer restoreUserStore()
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
				return model.TenantEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "11"}}, Name: "租户A", Status: model.TenantStatusSuspended, Tag: "a"},
				}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	_, err := svc.MyTenants(ginCtx, &dtoauth.MyTenantsReq{})
	assertCode(t, err, code.TenantSuspendedError)
}

// TestMyTenantsReportsMissingTenantRowsAsDataError 成员关系存在但租户行查不到属数据不一致，
// 必须报数据类错误而不是"该租户已被挂起"（否则掩盖真实根因）。
func TestMyTenantsReportsMissingTenantRowsAsDataError(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, TenantID: "11", PersonID: "88", Name: "tenant-user-a"},
				}, nil
			},
		}
	})
	defer restoreUserStore()
	// 租户列表查询返回空：成员关系指向的租户行不存在
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getListByCondFunc: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, error) {
				return model.TenantEntityList{}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	_, err := svc.MyTenants(ginCtx, &dtoauth.MyTenantsReq{})
	assertCode(t, err, code.UserGetDetailError)
}

func TestJoinTenantRejectsMissingInviteCode(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: ""})
	assertCode(t, err, code.AuthJoinNotAllowedError)
	if cerr := db.Exec("SELECT 1 FROM tenant_invite").Error; cerr != nil {
		t.Fatalf("expected invite table migrated: %v", cerr)
	}
}

func TestJoinTenantRejectsInvalidInvite(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "no-such-code"})
	assertCode(t, err, code.InviteInvalidError)
}

func TestJoinTenantRejectsRevokedInvite(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusRevoked, nil)

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	assertCode(t, err, code.InviteInvalidError)
}

// TestJoinTenantRejectsExpiredInvite 过期由 expires_at 在读取时派生，而非存储态（D6/R2）：
// 库里没有 "expired" 这个状态值，能否加入只取决于 expires_at 是否已过。
func TestJoinTenantRejectsExpiredInvite(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	past := time.Now().Add(-time.Hour)
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, &past)

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	assertCode(t, err, code.InviteExpiredError)
}

// TestJoinTenantAcceptsInviteNotYetExpired 未到期的邀请仍可加入——
// 防止把过期比较写反（ExpiresAt 在未来却被判过期）。
func TestJoinTenantAcceptsInviteNotYetExpired(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	future := time.Now().Add(time.Hour)
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, &future)

	svc := &authSvc{}
	resp, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	if err != nil {
		t.Fatalf("未过期邀请应可加入, got err: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created user id, got %#v", resp)
	}
}

func TestJoinTenantRejectsAlreadyJoinedTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)
	now := time.Now()
	existing := &model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}},
		TenantID:   "22",
		PersonID:   "88",
		Name:       "existing",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
		JoinedAt:   &now,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("seed existing user: %v", err)
	}

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	assertCode(t, err, code.AlreadyJoinedError)
}

func TestJoinTenantCreatesNonOwnerUser(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{}, &model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, true))
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)

	svc := &authSvc{}
	resp, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	if err != nil {
		t.Fatalf("JoinTenant returned error: %v", err)
	}
	if resp == nil || resp.UserID == "" {
		t.Fatalf("expected created user id, got %#v", resp)
	}

	var insertedUser model.UserEntity
	// 目标是邀请码解析出的租户 22：显式声明该租户作用域，与 JoinTenant 内部一致
	if err := db.WithContext(dbclient.ExplicitTenantContext(ginCtx, "22")).Where("id = ?", resp.UserID).First(&insertedUser).Error; err != nil {
		t.Fatalf("expected user persisted: %v", err)
	}
	if insertedUser.TenantID != "22" {
		t.Fatalf("expected tenant id 22, got %s", insertedUser.TenantID)
	}
	if insertedUser.PersonID != "88" {
		t.Fatalf("expected person id 88, got %s", insertedUser.PersonID)
	}
	if insertedUser.IsOwner {
		t.Fatalf("expected join-tenant user to be non-owner (isOwner=false), got %t", insertedUser.IsOwner)
	}
	if insertedUser.JoinedAt == nil {
		t.Fatal("expected join-tenant user to have joined_at set")
	}

	// 邀请应被标记为已使用；邀请码全局唯一，按 code 反查需显式声明跨租户作用域
	var invite model.InviteEntity
	if err := db.WithContext(dbclient.CrossTenantContext(ginCtx)).Where("code = ?", "invite-abc").First(&invite).Error; err != nil {
		t.Fatalf("expected invite persisted: %v", err)
	}
	if invite.Status != model.InviteStatusAccepted {
		t.Fatalf("expected invite marked accepted, got %s", invite.Status)
	}
}

// seedJoinByInviteApp 播种一个应用及其 OIDC 客户端，返回该客户端的 client_id。
//
// 通道 B（凭邀请加入租户）的应用级门禁按调用方 access token 的 client_id 解析应用
// （见 pkg/core/application），因此凡是要走到邀请校验之后的 JoinTenant 用例，都必须先
// 具备一个带 AllowJoinByInvite 的应用；allow=false 用于验证门禁关闭时的拒绝路径。
func seedJoinByInviteApp(t *testing.T, db *gorm.DB, allow bool) string {
	t.Helper()
	appEntity := &model.ApplicationEntity{Code: "app_join", AllowJoinByInvite: model.BoolPtr(allow)}
	if err := db.Create(appEntity).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	client := &model.ApplicationClientEntity{Code: "join_client", AppID: appEntity.ID}
	client.ID = client.AppID
	client.RedirectURIs = datatypes.JSON(`[]`)
	client.PostLogoutRedirectURIs = datatypes.JSON(`[]`)
	client.GrantTypes = datatypes.JSON(`["authorization_code"]`)
	client.ResponseTypes = datatypes.JSON(`["code"]`)
	client.AllowedOrigins = datatypes.JSON(`[]`)
	client.DefaultScopes = datatypes.JSON(`["openid"]`)
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed application client: %v", err)
	}
	return client.Code
}

// TestJoinTenantRejectsWhenAppDisallowsInvite 应用级门禁（通道 B）：
// 应用未开启 allow_join_by_invite 时，即使邀请本身完全有效也必须拒绝，
// 且门禁先于邀请解析——被拒时不得消费邀请。
func TestJoinTenantRejectsWhenAppDisallowsInvite(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{},
		&model.ApplicationEntity{}, &model.ApplicationClientEntity{})
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)
	ginCtx.Set(middleware.ContextKeyClientID, seedJoinByInviteApp(t, db, false))

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	assertCode(t, err, code.AuthJoinNotAllowedError)

	// 邀请码全局唯一，按 code 反查需显式声明跨租户作用域
	var invite model.InviteEntity
	if err := db.WithContext(dbclient.CrossTenantContext(ginCtx)).Where("code = ?", "invite-abc").First(&invite).Error; err != nil {
		t.Fatalf("expected invite persisted: %v", err)
	}
	if invite.Status != model.InviteStatusPending {
		t.Fatalf("门禁拒绝时邀请应保持 pending，实际为 %s", invite.Status)
	}
}

// TestJoinTenantRejectsWhenClientUnresolvable 解析不出应用（无 client_id、客户端或应用不存在，
// 例如 API Key 通道）一律 fail-closed，不得退化成放行。
func TestJoinTenantRejectsWhenClientUnresolvable(t *testing.T) {
	for _, c := range []struct{ name, clientID string }{
		{name: "empty client id", clientID: ""},
		{name: "unknown client id", clientID: "no_such_client"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ginCtx, _ := gin.CreateTestContext(nil)
			ginCtx.Request = httptestRequest(t)
			ginCtx.Set(gcontext.KeyPersonID, "88")
			ginCtx.Set(middleware.ContextKeyClientID, c.clientID)

			db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{},
				&model.ApplicationEntity{}, &model.ApplicationClientEntity{})
			seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)

			svc := &authSvc{}
			_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
			assertCode(t, err, code.AuthJoinNotAllowedError)
		})
	}
}

// seedInvite 向测试库播种一条邀请。
func seedInvite(t *testing.T, db *gorm.DB, code, tenantID string, status model.InviteStatus, expiresAt *time.Time) {
	t.Helper()
	invite := &model.InviteEntity{
		Code:      code,
		TenantID:  tenantID,
		Status:    status,
		ExpiresAt: expiresAt,
	}
	if err := db.Create(invite).Error; err != nil {
		t.Fatalf("seed invite: %v", err)
	}
}
