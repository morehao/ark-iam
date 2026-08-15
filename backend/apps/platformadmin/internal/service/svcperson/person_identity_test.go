package svcperson

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func newPersonIdentityGinCtx(tenantID, userID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

// seedUser 播种用户数据。SQLite 下 user.joined_at 为 NOT NULL 且显式 NULL
// 不会回退到默认值，必须显式设置 JoinedAt（profile/custom_data 同理）。
func seedUser(t *testing.T, db *gorm.DB, tenantID, personID string, name string) *model.UserEntity {
	t.Helper()
	now := time.Now()
	u := &model.UserEntity{
		TenantID:   tenantID,
		PersonID:   personID,
		Name:       name,
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		JoinedAt:   &now,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func TestPersonIdentityCreatePersistsPersonID(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	ctx := newPersonIdentityGinCtx("88", "501")

	// 自然人 66 在租户 88 下拥有用户，身份创建应落到该自然人
	seedUser(t, db, "88", "66", "alice")

	svc := &personSvc{}
	resp, err := svc.Create(ctx, &dtouser.UserIdentityCreateReq{
		TenantID:   "88",
		UserID:     "66",
		Issuer:     "https://issuer.example.com",
		IdentityID: "external-subject-1",
		Detail: map[string]any{
			"source": "oidc",
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if resp == nil || resp.UserIdentityID == "" {
		t.Fatalf("expected created identity response, got %#v", resp)
	}

	got, err := dao.NewUserIdentityDao().GetByID(ctx, resp.UserIdentityID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("expected identity persisted")
	}
	if got.PersonID != "66" {
		t.Fatalf("expected person_id 66, got %s", got.PersonID)
	}
	if got.Issuer != "https://issuer.example.com" {
		t.Fatalf("expected issuer to be persisted, got %q", got.Issuer)
	}
	if got.ExternalSubject != "external-subject-1" {
		t.Fatalf("expected external subject to be persisted, got %q", got.ExternalSubject)
	}
	if got.CreatedBy != "501" {
		t.Fatalf("expected created_by 501, got %s", got.CreatedBy)
	}
}

func TestPersonIdentityUpdateDoesNotPersistFakeUpdatedByWhenOperatorMissing(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	ctx := newPersonIdentityGinCtx("31", "")

	seedUser(t, db, "31", "71", "bob")
	identity := &model.UserIdentityEntity{PersonID: "71", Issuer: "issuer-old", ExternalSubject: "external-old", Detail: json.RawMessage(`{}`)}
	if err := db.Create(identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	svc := &personSvc{}
	err := svc.Update(ctx, &dtouser.UserIdentityUpdateReq{
		UserIdentityID: identity.ID,
		UserID:         "71",
		Issuer:         "issuer-a",
		IdentityID:     "external-a",
		Detail:         map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	got, err := dao.NewUserIdentityDao().GetByID(ctx, identity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID == "" {
		t.Fatalf("expected identity persisted")
	}
	if got.UpdatedBy != "" {
		t.Fatalf("expected updated_by to be omitted when operator missing, got %s", got.UpdatedBy)
	}
}

func TestPersonIdentityDetailRejectsCrossTenantPerson(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserIdentityEntity{}, &model.UserEntity{})
	ctx := newPersonIdentityGinCtx("31", "")

	// 该自然人只存在于租户 99，当前上下文租户 31 无权访问
	seedUser(t, db, "99", "71", "carol")
	identity := &model.UserIdentityEntity{PersonID: "71", Detail: json.RawMessage(`{}`)}
	if err := db.Create(identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	svc := &personSvc{}
	_, err := svc.Detail(ctx, &dtouser.UserIdentityDetailReq{UserIdentityID: identity.ID})
	if err == nil {
		t.Fatal("expected cross-tenant person identity detail to fail")
	}
}
