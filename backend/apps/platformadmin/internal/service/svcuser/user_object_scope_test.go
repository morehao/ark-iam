package svcuser

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

// testSeedUser 播种用户：sqlite 对 joined_at 的 not null 约束要求显式值。
func testSeedUser(db *gorm.DB, u *model.UserEntity) error {
	now := time.Now()
	if u.JoinedAt == nil {
		u.JoinedAt = &now
	}
	return db.Create(u).Error
}

func TestUserDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(81))

	db := testutil.SetupSQLite(t, &model.UserEntity{})
	if err := testSeedUser(db, &model.UserEntity{
		Model:      gorm.Model{ID: 7},
		TenantID:   99,
		PersonID:   1,
		Name:       "cross",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := &userSvc{}
	_, err := svc.Detail(ginCtx, &dtouser.UserDetailReq{UserID: 7})
	if err == nil {
		t.Fatalf("expected cross-tenant user detail to fail")
	}
}

func TestUserPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(82))

	db := testutil.SetupSQLite(t, &model.UserEntity{})
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:   82,
		PersonID:   1,
		Name:       "tenant-82",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed tenant 82: %v", err)
	}
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:   99,
		PersonID:   2,
		Name:       "tenant-99",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed tenant 99: %v", err)
	}

	svc := &userSvc{}
	// 请求里的 TenantID 应为 99，但服务必须以上下文租户 82 为准
	resp, err := svc.PageList(ginCtx, &dtouser.UserPageListReq{TenantID: 99})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1 (context tenant), got %d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].TenantID != 82 {
		t.Fatalf("expected tenant 82 user from context, got %+v", resp.List)
	}
}

func TestDetailUserLoginLogRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(83))

	db := testutil.SetupSQLite(t, &model.UserLoginLogEntity{})
	if err := db.Create(&model.UserLoginLogEntity{
		Model:    gorm.Model{ID: 8},
		TenantID: 91,
		UserID:   1,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := &userSvc{}
	_, err := svc.DetailUserLoginLog(ginCtx, &dtouser.UserLoginLogDetailReq{UserLoginLogID: 8})
	if err == nil {
		t.Fatalf("expected cross-tenant login log detail to fail")
	}
}

func TestUserIdentityDetailRejectsCrossTenantEntity(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(84))

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// 身份归属 person 92，而该 person 在租户 99 有用户（不在当前租户 84）
	if err := db.Create(&model.UserIdentityEntity{
		Model:           gorm.Model{ID: 9},
		PersonID:        92,
		Issuer:          "issuer-a",
		ExternalSubject: "external-a",
		Detail:          []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	if err := testSeedUser(db, &model.UserEntity{
		Model:      gorm.Model{ID: 9},
		TenantID:   99,
		PersonID:   92,
		Name:       "other-tenant",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := &userIdentitySvc{}
	_, err := svc.Detail(ginCtx, &dtouser.UserIdentityDetailReq{UserIdentityID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant identity detail to fail")
	}
}

func TestUserIdentityPageListUsesContextTenant(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(85))

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// person 99 在当前租户 85 有用户，可查询其身份
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:   85,
		PersonID:   99,
		Name:       "tenant-85",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        99,
		Issuer:          "issuer-a",
		ExternalSubject: "external-1",
		Detail:          []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	svc := &userIdentitySvc{}
	resp, err := svc.PageList(ginCtx, &dtouser.UserIdentityPageListReq{UserID: 99})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected 1 identity, got total=%d list=%d", resp.Total, len(resp.List))
	}
	if resp.List[0].UserID != 99 {
		t.Fatalf("expected identity for person 99, got %+v", resp.List[0])
	}
}
