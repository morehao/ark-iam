package svcauth

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func TestUserinfoReturnsPersonAndTenantUser(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyUserID, "21")
	ginCtx.Set(gcontext.KeyTenantID, "11")
	ginCtx.Set(gcontext.KeyPersonID, "101")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.UserEntity, error) {
				if id != "21" {
					t.Fatalf("expected user lookup by userID 21, got %s", id)
				}
				return &model.UserEntity{
					BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "21"}},
					TenantID:   "11",
					PersonID:   "101",
					Name:       "租户用户",
					IsOwner:    true,
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.PersonEntity, error) {
				return &model.PersonEntity{
					BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "101"}},
					Name:       "自然人",
				}, nil
			},
		}
	})
	defer restorePersonStore()

	svc := &authSvc{}
	resp, err := svc.Userinfo(ginCtx, &dtoauth.UserinfoReq{})
	if err != nil {
		t.Fatalf("Userinfo returned error: %v", err)
	}
	if resp.PersonInfo.PersonID != "101" || resp.PersonInfo.Name != "自然人" {
		t.Fatalf("unexpected personInfo: %+v", resp.PersonInfo)
	}
	if resp.UserInfo.UserID != "21" || resp.UserInfo.TenantID != "11" || resp.UserInfo.Name != "租户用户" || !resp.UserInfo.IsOwner {
		t.Fatalf("unexpected userInfo: %+v", resp.UserInfo)
	}
}

func TestUserinfoReturnsPersonFromContextWhenPersonMissing(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	ginCtx.Set(gcontext.KeyUserID, "21")
	ginCtx.Set(gcontext.KeyTenantID, "11")
	ginCtx.Set(gcontext.KeyPersonID, "99")

	restoreUserStore := swapUserStoreFactory(func() authUserStore {
		return &fakeAuthUserStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.UserEntity, error) {
				return &model.UserEntity{
					BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "21"}},
					TenantID:   "11",
					PersonID:   "99",
					Name:       "租户用户",
					IsOwner:    false,
				}, nil
			},
		}
	})
	defer restoreUserStore()

	restorePersonStore := swapPersonStoreFactory(func() authPersonStore {
		return &fakeAuthPersonStore{
			getByIDFunc: func(ctx context.Context, id string) (*model.PersonEntity, error) {
				return &model.PersonEntity{
					BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "99"}},
					Name:       "自然人99",
				}, nil
			},
		}
	})
	defer restorePersonStore()

	svc := &authSvc{}
	resp, err := svc.Userinfo(ginCtx, &dtoauth.UserinfoReq{})
	if err != nil {
		t.Fatalf("Userinfo returned error: %v", err)
	}
	if resp.PersonInfo.PersonID != "99" || resp.PersonInfo.Name != "自然人99" {
		t.Fatalf("expected personInfo from user's person_id, got %+v", resp.PersonInfo)
	}
	if resp.UserInfo.UserID != "21" || resp.UserInfo.IsOwner {
		t.Fatalf("expected userInfo populated, got %+v", resp.UserInfo)
	}
}
