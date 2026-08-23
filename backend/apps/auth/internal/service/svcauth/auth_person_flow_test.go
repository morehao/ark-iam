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
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
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
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "11"}}, Name: "租户A", Tag: "a"},
					{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "12"}}, Name: "租户B", Tag: "b"},
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

func TestJoinTenantRejectsMissingInviteCode(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{})

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

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{})
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusPending, nil)

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "no-such-code"})
	assertCode(t, err, code.InviteInvalidError)
}

func TestJoinTenantRejectsRevokedInvite(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{})
	seedInvite(t, db, "invite-abc", "22", model.InviteStatusRevoked, nil)

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{InviteCode: "invite-abc"})
	assertCode(t, err, code.InviteInvalidError)
}

func TestJoinTenantRejectsAlreadyJoinedTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{})
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

	db := testutil.SetupSQLite(t, &model.InviteEntity{}, &model.UserEntity{})
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
	if err := db.Where("id = ?", resp.UserID).First(&insertedUser).Error; err != nil {
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

	// 邀请应被标记为已使用
	var invite model.InviteEntity
	if err := db.Where("code = ?", "invite-abc").First(&invite).Error; err != nil {
		t.Fatalf("expected invite persisted: %v", err)
	}
	if invite.Status != model.InviteStatusAccepted {
		t.Fatalf("expected invite marked accepted, got %s", invite.Status)
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
