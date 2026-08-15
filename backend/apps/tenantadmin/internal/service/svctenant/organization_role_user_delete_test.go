package svctenant

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestDeleteOrganizationRoleUserUsesTenantScopedCompositeLookup(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationRoleUserEntity{})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(81))
	ginCtx.Set(gcontext.KeyUserID, uint(9401))

	relation := &model.OrganizationRoleUserEntity{
		Model:              gorm.Model{ID: 119},
		TenantID:           81,
		OrganizationID:     401,
		OrganizationRoleID: 501,
		UserID:             601,
	}
	if err := db.Create(relation).Error; err != nil {
		t.Fatalf("seed relation: %v", err)
	}

	svc := &organizationRoleUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserDeleteReq{OrganizationRoleID: 501, UserID: 601})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 租户作用域复合条件命中后，应按关系记录 id 软删除，并记录操作人
	left, err := dao.NewOrganizationRoleUserDao().GetListByCond(ginCtx, &dao.OrganizationRoleUserCond{
		TenantID:           81,
		OrganizationRoleID: 501,
		UserID:             601,
	})
	if err != nil {
		t.Fatalf("GetListByCond: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected relation soft-deleted, got %+v", left)
	}

	var deleted model.OrganizationRoleUserEntity
	if err := db.Unscoped().Where("id = ?", 119).First(&deleted).Error; err != nil {
		t.Fatalf("load deleted row: %v", err)
	}
	if !deleted.DeletedAt.Valid {
		t.Fatalf("expected deleted_at set")
	}
	if deleted.DeletedBy != 9401 {
		t.Fatalf("expected deletedBy 9401, got %d", deleted.DeletedBy)
	}
}

func TestDeleteOrganizationRoleUserReturnsNotExistWhenCompositeLookupMisses(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.OrganizationRoleUserEntity{})
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(82))
	ginCtx.Set(gcontext.KeyUserID, uint(9402))

	svc := &organizationRoleUserSvc{}
	err := svc.Delete(ginCtx, &dtotenant.OrganizationRoleUserDeleteReq{OrganizationRoleID: 501, UserID: 601})
	if err == nil {
		t.Fatalf("expected not exist error")
	}

	// 未命中任何关系记录时，不应发生删除
	var count int64
	if err := db.Model(&model.OrganizationRoleUserEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no rows after failed delete, got %d", count)
	}
}
