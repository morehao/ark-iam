package svctenantapplication

import (
	"testing"

	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
)

// seedAppEntityWithSource 播种指定来源的应用（内置/第三方），用于验证订阅删除的内置判定。
func seedAppEntityWithSource(t *testing.T, db *gorm.DB, appID, name string, source model.AppSource) {
	t.Helper()
	if err := db.Create(&model.ApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: appID}},
		Code:       "code-" + appID,
		Name:       name,
		Source:     source,
		Status:     model.AppStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed application %s: %v", appID, err)
	}
}

// seedTenantAppEntity 播种一条租户应用订阅。
func seedTenantAppEntity(t *testing.T, db *gorm.DB, tenantID, appID string) *model.TenantApplicationEntity {
	t.Helper()
	entity := &model.TenantApplicationEntity{
		TenantID: tenantID,
		AppID:    appID,
		Status:   model.TenantApplicationStatusEnable,
	}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed tenant_application(%s,%s): %v", tenantID, appID, err)
	}
	return entity
}

// TestDeleteBuiltInApplicationSubscriptionForbidden 内置应用（source=builtin）的订阅禁删：
// 种子为平台租户写入 platform_admin 订阅、ProvisionTenantAdmin 为每个租户写入 tenant_admin 订阅，
// 删除会让对应控制台当场失去菜单（平台侧整栈失联），且产品内无恢复路径。
func TestDeleteBuiltInApplicationSubscriptionForbidden(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t1", "租户一")
	seedAppEntityWithSource(t, db, "app-builtin", "平台管理后台", model.AppSourceBuiltin)
	sub := seedTenantAppEntity(t, db, "t1", "app-builtin")

	ctx := newCreateCtx()
	err := NewTenantApplicationSvc().Delete(ctx, &dtotenantapplication.TenantApplicationDeleteReq{TenantAppID: sub.ID})
	if err == nil {
		t.Fatal("内置应用的订阅必须拒绝删除")
	}
	if gerror.GetCode(err) != int(code.TenantApplicationBuiltInErr) {
		t.Fatalf("期望 TenantApplicationBuiltInErr, got %v", err)
	}

	// 被拒的删除不得落库（订阅仍可查到）
	got, err := dao.NewTenantApplicationDao().GetByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatal("被拒删除的订阅不得消失")
	}
}

// TestDeleteThirdPartyApplicationSubscription 第三方应用的订阅可删（删除人写入上下文用户）。
func TestDeleteThirdPartyApplicationSubscription(t *testing.T) {
	db := setupTenantAppDB(t)
	seedTenantEntity(t, db, "t1", "租户一")
	seedAppEntityWithSource(t, db, "app-third", "第三方应用", model.AppSourceThirdParty)
	sub := seedTenantAppEntity(t, db, "t1", "app-third")

	ctx := newCreateCtx()
	if err := NewTenantApplicationSvc().Delete(ctx, &dtotenantapplication.TenantApplicationDeleteReq{TenantAppID: sub.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	got, err := dao.NewTenantApplicationDao().GetByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("expected subscription soft-deleted, got %+v", got)
	}

	// 断言软删行用被测服务同一个租户作用域（订阅归属 t1，非跨租户查询）
	var deleted model.TenantApplicationEntity
	if err := db.WithContext(ctx).Unscoped().Where("id = ?", sub.ID).First(&deleted).Error; err != nil {
		t.Fatalf("query deleted row: %v", err)
	}
	if deleted.DeletedBy != "u1" {
		t.Fatalf("expected deletedBy u1, got %s", deleted.DeletedBy)
	}
}
