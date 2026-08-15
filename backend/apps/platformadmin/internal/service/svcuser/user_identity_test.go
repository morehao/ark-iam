package svcuser

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

func TestUserIdentityGetByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, "45")

	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	// 用户 202（租户 45）关联 person 302，身份按 person 302 查询
	if err := testSeedUser(db, &model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "202"}},
		TenantID:   "45",
		PersonID:   "302",
		Name:       "user-202",
		Profile:    []byte("{}"),
		CustomData: []byte("{}"),
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        "302",
		Issuer:          "issuer-b",
		ExternalSubject: "external-2",
		Detail:          []byte(`{"name":"second"}`),
	}).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	// 其他 person 的身份不应返回
	if err := db.Create(&model.UserIdentityEntity{
		PersonID:        "999",
		Issuer:          "issuer-x",
		ExternalSubject: "external-x",
		Detail:          []byte(`{}`),
	}).Error; err != nil {
		t.Fatalf("seed other identity: %v", err)
	}

	svc := &userIdentitySvc{}
	resp, err := svc.GetByUser(ginCtx, &dtouser.UserIdentityByUserReq{UserID: "202"})
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("expected 1 identity for person 302, got total=%d list=%d", resp.Total, len(resp.List))
	}
	if resp.List[0].UserID != "302" {
		t.Fatalf("expected identity mapped to person 302, got %+v", resp.List[0])
	}
}

