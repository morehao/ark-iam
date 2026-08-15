package svcuser

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func TestGetUserLoginLogByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(45))

	db := testutil.SetupSQLite(t, &model.UserLoginLogEntity{})
	// 同用户跨租户的两条日志：只有本租户的应被返回
	if err := db.Create(&model.UserLoginLogEntity{
		TenantID:  45,
		UserID:    202,
		LoginIP:   "127.0.0.1",
		UserAgent: "chrome",
	}).Error; err != nil {
		t.Fatalf("seed tenant 45 log: %v", err)
	}
	if err := db.Create(&model.UserLoginLogEntity{
		TenantID:  46,
		UserID:    202,
		LoginIP:   "10.0.0.1",
		UserAgent: "firefox",
	}).Error; err != nil {
		t.Fatalf("seed tenant 46 log: %v", err)
	}

	svc := &userSvc{}
	resp, err := svc.GetUserLoginLogByUser(ginCtx, &dtouser.UserLoginLogByUserReq{UserID: 202})
	if err != nil {
		t.Fatalf("GetUserLoginLogByUser returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1 (tenant scoped), got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
	if resp.List[0].TenantID != 45 || resp.List[0].UserID != 202 {
		t.Fatalf("expected tenant 45 log, got %+v", resp.List[0])
	}
}

func TestGetUserDepartmentByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(46))

	db := testutil.SetupSQLite(t, &model.UserDepartmentEntity{})
	// 同用户跨租户的两条部门关系：只有本租户的应被返回
	if err := db.Create(&model.UserDepartmentEntity{
		TenantID:     46,
		UserID:       303,
		DepartmentID: 404,
		IsPrimary:    1,
	}).Error; err != nil {
		t.Fatalf("seed tenant 46 relation: %v", err)
	}
	if err := db.Create(&model.UserDepartmentEntity{
		TenantID:     47,
		UserID:       303,
		DepartmentID: 505,
		IsPrimary:    0,
	}).Error; err != nil {
		t.Fatalf("seed tenant 47 relation: %v", err)
	}

	svc := &userSvc{}
	resp, err := svc.GetUserDepartmentByUser(ginCtx, &dtouser.UserDepartmentByUserReq{UserID: 303})
	if err != nil {
		t.Fatalf("GetUserDepartmentByUser returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1 (tenant scoped), got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
	if resp.List[0].TenantID != 46 || resp.List[0].UserID != 303 || resp.List[0].DepartmentID != 404 {
		t.Fatalf("expected tenant 46 relation, got %+v", resp.List[0])
	}
}
