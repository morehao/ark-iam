package svcauth

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type fakeAuthPersonStore struct {
	getByIDFunc   func(ctx context.Context, id uint) (*model.PersonEntity, error)
	getByCondFunc func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error)
	insertFunc    func(ctx context.Context, entity *model.PersonEntity) error
}

func (f *fakeAuthPersonStore) GetByID(ctx context.Context, id uint) (*model.PersonEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthPersonStore) GetByCond(ctx context.Context, cond genericdao.Cond) (*model.PersonEntity, error) {
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
	getByIDFunc       func(ctx context.Context, id uint) (*model.TenantEntity, error)
	getPageListByCond func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, int64, error)
}

func (f *fakeAuthTenantStore) GetByID(ctx context.Context, id uint) (*model.TenantEntity, error) {
	if f.getByIDFunc == nil {
		return nil, nil
	}
	return f.getByIDFunc(ctx, id)
}

func (f *fakeAuthTenantStore) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.TenantEntityList, int64, error) {
	if f.getPageListByCond == nil {
		return nil, 0, nil
	}
	tenantCond, _ := cond.(*dao.TenantCond)
	return f.getPageListByCond(ctx, tenantCond)
}

func TestLoginAuthenticatesByPersonWithoutTenantID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	var personLookup *dao.PersonCond
	var userLookup *dao.UserCond
	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				personLookup = cond
				return &model.PersonEntity{
					Model:             gorm.Model{ID: 88},
					PrimaryEmail:      "person@example.com",
					PasswordEncrypted: mustGeneratePasswordHash(t, "Password1"),
					Name:              "person-name",
				}, nil
			},
		}
	})
	defer restorePersonStore()

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				userLookup = cond
				return &model.UserEntity{
					Model:    gorm.Model{ID: 101},
					TenantID: 22,
					PersonID: 88,
					Name:     "tenant-user",
				}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "tenant-user"}}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				if id != 22 {
					return nil, nil
				}
				return &model.TenantEntity{Model: gorm.Model{ID: 22}, Name: "租户A", Tag: "tenant-a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{}
	})
	defer restoreRefreshTokenStore()
	restoreLoginRecorder := swapLoginRecorder(func(ctx *gin.Context, tenantID, userID uint, success bool) {})
	defer restoreLoginRecorder()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Login(ginCtx, &dtoauth.LoginReq{Identifier: "person@example.com", Password: "Password1"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if personLookup == nil {
		t.Fatal("expected login to query person")
	}
	if personLookup.PrimaryEmail != "person@example.com" {
		t.Fatalf("expected person lookup by email, got %+v", *personLookup)
	}
	if userLookup == nil {
		t.Fatal("expected login to query tenant user by person")
	}
	if userLookup.PersonID != 88 {
		t.Fatalf("expected user lookup by personID 88, got %+v", *userLookup)
	}
	if userLookup.TenantID != 0 {
		t.Fatalf("expected login not to require tenantID, got %+v", *userLookup)
	}
	if resp == nil || resp.PersonToken.AccessToken == "" {
		t.Fatalf("expected person token in login response, got %#v", resp)
	}
	if len(resp.Tenants) != 1 || resp.Tenants[0].TenantID != 22 {
		t.Fatalf("expected login response to include tenant list, got %#v", resp)
	}
}

func TestLoginReturnsOnlyJoinedTenants(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)

	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByCondFunc: func(ctx context.Context, cond *dao.PersonCond) (*model.PersonEntity, error) {
				return &model.PersonEntity{
					Model:             gorm.Model{ID: 88},
					Username:          "person-user",
					PasswordEncrypted: mustGeneratePasswordHash(t, "Password1"),
				}, nil
			},
		}
	})
	defer restorePersonStore()

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				if cond.PersonID != 88 {
					t.Fatalf("unexpected person lookup cond: %+v", cond)
				}
				return &model.UserEntity{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "tenant-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				if cond.PersonID != 88 {
					t.Fatalf("unexpected joined tenant cond: %+v", cond)
				}
				return model.UserEntityList{
					{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "tenant-user-a"},
					{Model: gorm.Model{ID: 102}, TenantID: 33, PersonID: 88, Name: "tenant-user-b"},
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				switch id {
				case 22:
					return &model.TenantEntity{Model: gorm.Model{ID: 22}, Name: "租户A", Tag: "a"}, nil
				case 33:
					return &model.TenantEntity{Model: gorm.Model{ID: 33}, Name: "租户B", Tag: "b"}, nil
				default:
					return nil, nil
				}
			},
		}
	})
	defer restoreTenantStore()

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{}
	})
	defer restoreRefreshTokenStore()
	restoreLoginRecorder := swapLoginRecorder(func(ctx *gin.Context, tenantID, userID uint, success bool) {})
	defer restoreLoginRecorder()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.Login(ginCtx, &dtoauth.LoginReq{Identifier: "person-user", Password: "Password1"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if resp == nil || len(resp.Tenants) != 2 {
		t.Fatalf("expected exactly joined tenants, got %#v", resp)
	}
	if resp.Tenants[0].TenantID != 22 || resp.Tenants[1].TenantID != 33 {
		t.Fatalf("expected joined tenant IDs [22 33], got %#v", resp.Tenants)
	}
}

func TestMyTenantsReturnsCurrentPersonTenantList(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	var userLookup *dao.UserCond
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				userLookup = cond
				return &model.UserEntity{Model: gorm.Model{ID: 101}, TenantID: 11, PersonID: 88, Name: "tenant-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				userLookup = cond
				return model.UserEntityList{
					{Model: gorm.Model{ID: 101}, TenantID: 11, PersonID: 88, Name: "tenant-user-a"},
					{Model: gorm.Model{ID: 102}, TenantID: 12, PersonID: 88, Name: "tenant-user-b"},
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				switch id {
				case 11:
					return &model.TenantEntity{Model: gorm.Model{ID: 11}, Name: "租户A", Tag: "a"}, nil
				case 12:
					return &model.TenantEntity{Model: gorm.Model{ID: 12}, Name: "租户B", Tag: "b"}, nil
				default:
					return nil, nil
				}
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.MyTenants(ginCtx, &dtoauth.MyTenantsReq{})
	if err != nil {
		t.Fatalf("MyTenants returned error: %v", err)
	}
	if userLookup == nil || userLookup.PersonID != 88 {
		t.Fatalf("expected tenant lookup to use personID 88, got %+v", userLookup)
	}
	if resp == nil || len(resp.List) != 2 {
		t.Fatalf("expected two tenants, got %#v", resp)
	}
	if resp.List[0].TenantID != 11 || resp.List[1].TenantID != 12 {
		t.Fatalf("expected joined tenant IDs [11 12], got %#v", resp.List)
	}
}

func TestSelectTenantReturnsTenantScopedToken(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				if cond.PersonID != 88 || cond.TenantID != 22 {
					t.Fatalf("unexpected user lookup cond: %+v", *cond)
				}
				return &model.UserEntity{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "tenant-user"}, nil
			},
			getListByCondFunc: func(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
				return model.UserEntityList{{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "tenant-user"}}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: id}, Name: "租户A", Tag: "tenant-a"}, nil
			},
			getPageListByCond: func(ctx context.Context, cond *dao.TenantCond) (model.TenantEntityList, int64, error) {
				return model.TenantEntityList{{Model: gorm.Model{ID: 22}, Name: "租户A", Tag: "tenant-a"}}, 1, nil
			},
		}
	})
	defer restoreTenantStore()

	restoreRefreshTokenStore := swapRefreshTokenStoreFactory(func() authRefreshTokenStore {
		return &fakeAuthRefreshTokenStore{}
	})
	defer restoreRefreshTokenStore()
	restoreLoginRecorder := swapLoginRecorder(func(ctx *gin.Context, tenantID, userID uint, success bool) {})
	defer restoreLoginRecorder()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.SelectTenant(ginCtx, &dtoauth.SelectTenantReq{TenantID: 22})
	if err != nil {
		t.Fatalf("SelectTenant returned error: %v", err)
	}
	if resp == nil || resp.TokenInfo.AccessToken == "" {
		t.Fatalf("expected tenant token response, got %#v", resp)
	}
}

func TestSwitchTenantRejectsUnjoinedTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				if cond.PersonID != 88 || cond.TenantID != 99 {
					t.Fatalf("unexpected user lookup cond: %+v", *cond)
				}
				return nil, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.SwitchTenant(ginCtx, &dtoauth.SwitchTenantReq{TenantID: 99})
	assertCode(t, err, code.UserNotExistError)
}

func TestJoinTenantRejectsNonExistentTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
		}
	})
	defer restoreUserStore()

	var tenantLookup uint
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				tenantLookup = id
				return nil, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: 999})
	assertCode(t, err, code.TenantNotExistError)
	if tenantLookup != 999 {
		t.Fatalf("expected tenant lookup with id 999, got %d", tenantLookup)
	}
}

func TestJoinTenantRejectsAlreadyJoinedTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				if cond.PersonID != 88 || cond.TenantID != 22 {
					t.Fatalf("unexpected user lookup cond: %+v", *cond)
				}
				return &model.UserEntity{Model: gorm.Model{ID: 101}, TenantID: 22, PersonID: 88, Name: "existing"}, nil
			},
		}
	})
	defer restoreUserStore()

	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				return &model.TenantEntity{Model: gorm.Model{ID: 22}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	_, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: 22})
	assertCode(t, err, code.AlreadyJoinedError)
}

func TestJoinTenantCreatesNonOwnerUser(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyPersonID, uint(88))

	var insertedUser *model.UserEntity
	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByCondFunc: func(ctx context.Context, cond *dao.UserCond) (*model.UserEntity, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, entity *model.UserEntity) error {
				entity.ID = 200
				copied := *entity
				insertedUser = &copied
				return nil
			},
		}
	})
	defer restoreUserStore()

	var tenantLookup uint
	restoreTenantStore := swapTenantStoreFactory(func() authTenantStore {
		return &fakeAuthTenantStore{
			getByIDFunc: func(ctx context.Context, id uint) (*model.TenantEntity, error) {
				tenantLookup = id
				return &model.TenantEntity{Model: gorm.Model{ID: 22}, Name: "租户A", Tag: "a"}, nil
			},
		}
	})
	defer restoreTenantStore()

	svc := &authSvc{jwtSecret: "test-secret"}
	resp, err := svc.JoinTenant(ginCtx, &dtoauth.JoinTenantReq{TenantID: 22})
	if err != nil {
		t.Fatalf("JoinTenant returned error: %v", err)
	}
	if resp == nil || resp.UserID != 200 {
		t.Fatalf("expected user id 200, got %#v", resp)
	}
	if tenantLookup != 22 {
		t.Fatalf("expected tenant lookup with id 22, got %d", tenantLookup)
	}
	if insertedUser == nil {
		t.Fatal("expected user to be inserted")
	}
	if insertedUser.TenantID != 22 {
		t.Fatalf("expected tenant id 22, got %d", insertedUser.TenantID)
	}
	if insertedUser.PersonID != 88 {
		t.Fatalf("expected person id 88, got %d", insertedUser.PersonID)
	}
	if insertedUser.IsOwner != 0 {
		t.Fatalf("expected join-tenant user to be non-owner (isOwner=0), got %d", insertedUser.IsOwner)
	}
	if insertedUser.Name != "" {
		t.Fatalf("expected empty name for join-tenant user (person name not copied), got %q", insertedUser.Name)
	}
	if insertedUser.JoinedAt == nil {
		t.Fatal("expected join-tenant user to have joined_at set")
	}
}
