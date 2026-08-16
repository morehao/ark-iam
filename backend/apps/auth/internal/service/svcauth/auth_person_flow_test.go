package svcauth

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
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

func TestJoinTenantRejectsNonExistentTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
		}
	})
	defer restoreUserStore()

	var tenantLookup string
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
				tenantLookup = id
				return nil, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: "999"})
	assertCode(t, err, code.TenantNotExistError)
	if tenantLookup != "999" {
		t.Fatalf("expected tenant lookup with id 999, got %s", tenantLookup)
	}
}

func TestJoinTenantRejectsAlreadyJoinedTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				if cond.PersonID != "88" || cond.TenantID != "22" {
					t.Fatalf("unexpected user lookup cond: %+v", *cond)
				}
				return &model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}}, TenantID: "22", PersonID: "88", Name: "existing"}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
				return &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: "22"})
	assertCode(t, err, code.AlreadyJoinedError)
}

func TestJoinTenantCreatesNonOwnerUser(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, "88")

	var insertedUser *model.UserEntity
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = "200"
				copied := *entity
				insertedUser = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()

	var tenantLookup string
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.TenantEntity, error) {
				tenantLookup = id
				return &model.TenantEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "22"}}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{}
	resp, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: "22"})
	if err != nil {
		t.Fatalf("JoinTenant returned error: %v", err)
	}
	if resp == nil || resp.UserID != "200" {
		t.Fatalf("expected user id 200, got %#v", resp)
	}
	if tenantLookup != "22" {
		t.Fatalf("expected tenant lookup with id 22, got %s", tenantLookup)
	}
	if insertedUser == nil {
		t.Fatal("expected user to be inserted")
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
	if insertedUser.Name != "" {
		t.Fatalf("expected empty name for join-tenant user (person name not copied), got %q", insertedUser.Name)
	}
	if insertedUser.JoinedAt == nil {
		t.Fatal("expected join-tenant user to have joined_at set")
	}
}
