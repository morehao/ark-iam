package svctenant

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestOrganizationDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(41))

	if err := db.Create(&model.OrganizationEntity{Model: gorm.Model{ID: 8}, TenantID: 99, CustomData: json.RawMessage("{}")}).Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}

	svc := &organizationSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.OrganizationDetailReq{OrganizationID: 8})
	if err == nil {
		t.Fatalf("expected cross-tenant organization detail to fail, resp=%+v", resp)
	}
}

func TestOrganizationRoleDetailRejectsCrossTenantEntity(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationRoleEntity{})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(51))

	if err := db.Create(&model.OrganizationRoleEntity{Model: gorm.Model{ID: 9}, TenantID: 98}).Error; err != nil {
		t.Fatalf("seed organization role: %v", err)
	}

	svc := &organizationRoleSvc{}
	resp, err := svc.Detail(ginCtx, &dtotenant.OrganizationRoleDetailReq{OrganizationRoleID: 9})
	if err == nil {
		t.Fatalf("expected cross-tenant organization role detail to fail, resp=%+v", resp)
	}
}

func TestOrganizationRoleCreateRejectsCrossTenantOrganization(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationEntity{}, &model.OrganizationRoleEntity{})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(52))
	ginCtx.Set(gcontext.KeyUserID, uint(1001))

	if err := db.Create(&model.OrganizationEntity{Model: gorm.Model{ID: 6}, TenantID: 77, CustomData: json.RawMessage("{}")}).Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}

	svc := &organizationRoleSvc{}
	resp, err := svc.Create(ginCtx, &dtotenant.OrganizationRoleCreateReq{TenantID: 52, OrganizationID: 6, Name: "ops"})
	if err == nil {
		t.Fatalf("expected cross-tenant organization create to fail, resp=%+v", resp)
	}

	// 组织跨租户时不应插入角色记录
	var count int64
	if err := db.Model(&model.OrganizationRoleEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count roles: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no role inserted, got %d", count)
	}
}
