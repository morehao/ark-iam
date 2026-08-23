package svcuser

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

func TestUpdateOwnerGrantsAndRevokesTenantOwner(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyUserID, "1")

	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{})
	now := time.Now()
	user := &model.UserEntity{
		TenantID:   "22",
		PersonID:   "88",
		Name:       "member",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
		JoinedAt:   &now,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := &userSvc{}
	// 指派 owner
	if err := svc.UpdateOwner(ginCtx, &dtouser.UserOwnerUpdateReq{UserID: user.ID, IsOwner: true}); err != nil {
		t.Fatalf("UpdateOwner grant returned error: %v", err)
	}
	var afterGrant model.UserEntity
	if err := db.Where("id = ?", user.ID).First(&afterGrant).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !afterGrant.IsOwner {
		t.Fatalf("expected user to be owner after grant, got isOwner=%t", afterGrant.IsOwner)
	}

	// 取消 owner
	if err := svc.UpdateOwner(ginCtx, &dtouser.UserOwnerUpdateReq{UserID: user.ID, IsOwner: false}); err != nil {
		t.Fatalf("UpdateOwner revoke returned error: %v", err)
	}
	var afterRevoke model.UserEntity
	if err := db.Where("id = ?", user.ID).First(&afterRevoke).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if afterRevoke.IsOwner {
		t.Fatalf("expected user to NOT be owner after revoke, got isOwner=%t", afterRevoke.IsOwner)
	}
}

func TestUpdateOwnerRejectsNonExistentUser(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyUserID, "1")

	db := testutil.SetupSQLite(t, &model.UserEntity{})
	_ = db

	svc := &userSvc{}
	err := svc.UpdateOwner(ginCtx, &dtouser.UserOwnerUpdateReq{UserID: "no-such", IsOwner: true})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}
