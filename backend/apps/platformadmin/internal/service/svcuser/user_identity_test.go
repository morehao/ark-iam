package svcuser

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gobject"
	"gorm.io/gorm"
)

func TestUserIdentityPageListPassesFiltersToDAOAndKeepsDAOTotal(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// person 101 在当前租户有用户，可查询其身份
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:   23,
		PersonID:   101,
		Name:       "tenant-23",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// 命中过滤条件的一条 + 不命中的一条
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        101,
		Issuer:          "issuer-a",
		ExternalSubject: "external-1",
		Detail:          []byte(`{"name":"first"}`),
	}).Error; err != nil {
		t.Fatalf("seed matching identity: %v", err)
	}
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        101,
		Issuer:          "issuer-b",
		ExternalSubject: "external-2",
		Detail:          []byte(`{"name":"second"}`),
	}).Error; err != nil {
		t.Fatalf("seed other identity: %v", err)
	}

	svc := &userIdentitySvc{}
	resp, err := svc.PageList(ginCtx, &dtouser.UserIdentityPageListReq{
		PageQuery:  gobject.PageQuery{Page: 1, PageSize: 5},
		UserID:     101,
		Issuer:     "issuer-a",
		IdentityID: "external-1",
	})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected 1 filtered identity, got total=%d list=%d", resp.Total, len(resp.List))
	}
	item := resp.List[0]
	if item.Issuer != "issuer-a" || item.IdentityID != "external-1" || item.UserID != 101 {
		t.Fatalf("unexpected identity item: %+v", item)
	}
}

func TestUserIdentityGetByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(45))

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// 用户 202（租户 45）关联 person 302，身份按 person 302 查询
	if err := testSeedUser(db, &model.UserEntity{
		Model:      gorm.Model{ID: 202},
		TenantID:   45,
		PersonID:   302,
		Name:       "user-202",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        302,
		Issuer:          "issuer-b",
		ExternalSubject: "external-2",
		Detail:          []byte(`{"name":"second"}`),
	}).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	// 其他 person 的身份不应返回
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        999,
		Issuer:          "issuer-x",
		ExternalSubject: "external-x",
		Detail:          []byte(`{}`),
	}).Error; err != nil {
		t.Fatalf("seed other identity: %v", err)
	}

	svc := &userIdentitySvc{}
	resp, err := svc.GetByUser(ginCtx, &dtouser.UserIdentityByUserReq{UserID: 202})
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected 1 identity for person 302, got total=%d list=%d", resp.Total, len(resp.List))
	}
	if resp.List[0].UserID != 302 {
		t.Fatalf("expected identity mapped to person 302, got %+v", resp.List[0])
	}
}

func TestUserIdentityDetailUsesTenantUserToResolvePerson(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(84))

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// 用户 9（租户 84）关联 person 902，身份 11 归属 person 902
	if err := testSeedUser(db, &model.UserEntity{
		Model:      gorm.Model{ID: 9},
		TenantID:   84,
		PersonID:   902,
		Name:       "user-9",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserIdentityEntity{
		Model:           gorm.Model{ID: 11},
		PersonID:        902,
		Issuer:          "issuer-a",
		ExternalSubject: "external-a",
		Detail:          []byte(`{"name":"mapped"}`),
	}).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	svc := &userIdentitySvc{}
	resp, err := svc.Detail(ginCtx, &dtouser.UserIdentityDetailReq{UserIdentityID: 11})
	if err != nil {
		t.Fatalf("Detail returned error: %v", err)
	}
	if resp == nil || resp.UserID != 902 {
		t.Fatalf("expected mapped person response (UserID=902), got %#v", resp)
	}
}
