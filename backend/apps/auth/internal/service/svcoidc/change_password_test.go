package svcoidc

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"gorm.io/gorm"
)

// newChangePasswordEnv 构造"临时密码首次登录改密"链路所需的 OP provider + 内存库。
// 库中 person 承载口令与强制改密标记，refresh_token 供改密后的全局登出落库。
func newChangePasswordEnv(t *testing.T) (*OIDCProvider, *gorm.DB) {
	t.Helper()
	testsetup.Initialize(testsetup.AppNameAuth)
	t.Cleanup(func() { testsetup.Done(testsetup.AppNameAuth) })
	appconfig.Conf = &pkgconfig.Config{
		JWT: pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{
			Issuer:           "http://localhost:8099/oidc",
			FrontendLoginURL: "http://localhost:4000/oidc/login",
			AllowInsecure:    true,
		},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	db := testutil.SetupSQLite(t, &model.PersonEntity{}, &model.RefreshTokenEntity{})
	return provider, db
}

func seedTempPasswordPerson(t *testing.T, db *gorm.DB, tempPassword string) *model.PersonEntity {
	t.Helper()
	hash, err := gcrypto.GeneratePasswordHash(tempPassword)
	if err != nil {
		t.Fatalf("GeneratePasswordHash failed: %v", err)
	}
	person := &model.PersonEntity{
		Username:           model.StrPtr("temp-user"),
		PrimaryEmail:       model.StrPtr("temp@example.com"),
		PasswordEncrypted:  hash,
		PasswordMethod:     "bcrypt",
		MustChangePassword: true,
		Name:               "临时口令用户",
		Profile:            json.RawMessage(`{}`),
		CustomData:         json.RawMessage(`{}`),
	}
	if err := db.Create(person).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	return person
}

func newOIDCGinCtx(t *testing.T, path string) *gin.Context {
	t.Helper()
	ginCtx, _ := gin.CreateTestContext(nil)
	req, err := http.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ginCtx.Request = req
	return ginCtx
}

// TestCompleteLoginTempPasswordRequiresChange 持临时密码的自然人登录时：
// 不完成授权（done=false）以便用户先改密、不发 code、不建 SSO 会话，只回传 RequiresPasswordChange
// 并把 subject 绑到授权票据（改密接口据此识别身份）。
func TestCompleteLoginTempPasswordRequiresChange(t *testing.T) {
	testsetup.Initialize(testsetup.AppNameAuth)
	defer testsetup.Done(testsetup.AppNameAuth)
	appconfig.Conf = &pkgconfig.Config{
		JWT:  pkgconfig.JWT{SignKey: "test-sign-key"},
		OIDC: pkgconfig.OIDC{Issuer: "http://localhost:8099/oidc", AllowInsecure: true},
	}
	provider, err := SetupOIDCProvider()
	if err != nil {
		t.Fatalf("SetupOIDCProvider failed: %v", err)
	}
	authReq := newAuthReq(t, provider, "client-1")

	svc := &oidcAuthSvc{
		provider: provider,
		authSvc: &fakePasswordAuthenticator{authenticate: func(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
			return &model.PersonEntity{
					BaseEntity:         gormdao.BaseEntity{StringID: gormdao.StringID{ID: "88"}},
					MustChangePassword: true,
				},
				&model.UserEntity{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "66"}}, TenantID: "1", PersonID: "88"},
				[]objauth.TenantOption{{TenantID: "1", Name: "tenant-1"}},
				nil
		}},
	}

	res, err := svc.CompleteLogin(newOIDCGinCtx(t, "/oidc/login"), &dtooidc.OIDCLoginReq{
		AuthRequestID: authReq.GetID(),
		Identifier:    "temp@example.com",
		Password:      "Temp1234",
	})
	if err != nil {
		t.Fatalf("CompleteLogin failed: %v", err)
	}
	if !res.RequiresPasswordChange {
		t.Fatal("expected requiresPasswordChange=true for temporary password")
	}
	if res.ContinueURL != "" {
		t.Errorf("continueURL = %q, want empty before password change", res.ContinueURL)
	}
	if res.TenantID != "" {
		t.Errorf("tenantID = %q, want empty before password change", res.TenantID)
	}
	if res.SessionID != "" {
		t.Errorf("sessionID = %q, want empty before password change", res.SessionID)
	}
	if res.PersonID != "88" {
		t.Errorf("personID = %q, want 88", res.PersonID)
	}
	updated, err := provider.Storage.AuthRequestByID(t.Context(), authReq.GetID())
	if err != nil {
		t.Fatalf("AuthRequestByID failed: %v", err)
	}
	if updated.Done() {
		t.Fatal("auth request must stay incomplete until the password is changed")
	}
	if updated.GetSubject() != oidcop.BuildSubject("88") {
		t.Fatalf("subject = %q, want %q", updated.GetSubject(), oidcop.BuildSubject("88"))
	}
}

// TestChangePasswordRotatesAndClearsFlag 改密成功：口令哈希更新为新密码、强制改密标记清除、旧密码失效。
func TestChangePasswordRotatesAndClearsFlag(t *testing.T) {
	provider, db := newChangePasswordEnv(t)
	person := seedTempPasswordPerson(t, db, "Temp1234")
	authReq := newAuthReq(t, provider, "client-1")
	if err := provider.Storage.CompleteAuthRequest(t.Context(), authReq.GetID(),
		oidcop.BuildSubject(person.ID), time.Now(), []string{"pwd"}, "", "", false); err != nil {
		t.Fatalf("bind subject failed: %v", err)
	}

	svc := &oidcAuthSvc{provider: provider}
	err := svc.ChangePassword(newOIDCGinCtx(t, "/oidc/login/changePassword"), &dtooidc.OIDCChangePasswordReq{
		AuthRequestID:   authReq.GetID(),
		CurrentPassword: "Temp1234",
		NewPassword:     "Fresh1234",
	})
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	stored, err := dao.NewPersonDao().GetByID(t.Context(), person.ID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil {
		t.Fatal("person not found after change password")
	}
	if err := gcrypto.ComparePasswordHash(stored.PasswordEncrypted, "Fresh1234"); err != nil {
		t.Errorf("new password does not match stored hash: %v", err)
	}
	if err := gcrypto.ComparePasswordHash(stored.PasswordEncrypted, "Temp1234"); err == nil {
		t.Error("temporary password must no longer be valid")
	}
	if stored.MustChangePassword {
		t.Error("must_change_password must be cleared after a successful change")
	}
}

// TestChangePasswordRejections 改密前置校验：临时密码不匹配 / 新密码不满足强度 /
// 新密码与当前密码相同 / 该自然人并未处于强制改密状态，一律拒绝且不改动口令。
func TestChangePasswordRejections(t *testing.T) {
	cases := []struct {
		name            string
		mustChange      bool
		currentPassword string
		newPassword     string
		wantErr         int
	}{
		{name: "非强制改密不可经此端点改密", mustChange: false, currentPassword: "Temp1234", newPassword: "Fresh1234", wantErr: code.OIDCSessionNotFound},
		{name: "当前密码错误", mustChange: true, currentPassword: "Wrong1234", newPassword: "Fresh1234", wantErr: code.PasswordMismatchError},
		{name: "新密码不满足强度", mustChange: true, currentPassword: "Temp1234", newPassword: "weak", wantErr: code.PasswordValidationError},
		{name: "新旧密码相同", mustChange: true, currentPassword: "Temp1234", newPassword: "Temp1234", wantErr: code.PasswordValidationError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider, db := newChangePasswordEnv(t)
			person := seedTempPasswordPerson(t, db, "Temp1234")
			if !tc.mustChange {
				if err := db.Model(&model.PersonEntity{}).Where("id = ?", person.ID).
					Update("must_change_password", false).Error; err != nil {
					t.Fatalf("clear must_change_password: %v", err)
				}
			}
			authReq := newAuthReq(t, provider, "client-1")
			if err := provider.Storage.CompleteAuthRequest(t.Context(), authReq.GetID(),
				oidcop.BuildSubject(person.ID), time.Now(), []string{"pwd"}, "", "", false); err != nil {
				t.Fatalf("bind subject failed: %v", err)
			}

			svc := &oidcAuthSvc{provider: provider}
			err := svc.ChangePassword(newOIDCGinCtx(t, "/oidc/login/changePassword"), &dtooidc.OIDCChangePasswordReq{
				AuthRequestID:   authReq.GetID(),
				CurrentPassword: tc.currentPassword,
				NewPassword:     tc.newPassword,
			})
			if err == nil || err.Error() != code.GetError(tc.wantErr).Error() {
				t.Fatalf("ChangePassword err = %v, want %v", err, code.GetError(tc.wantErr))
			}

			stored, gErr := dao.NewPersonDao().GetByID(t.Context(), person.ID)
			if gErr != nil || stored == nil {
				t.Fatalf("dao GetByID failed, err:%v", gErr)
			}
			if err := gcrypto.ComparePasswordHash(stored.PasswordEncrypted, "Temp1234"); err != nil {
				t.Errorf("password must stay unchanged on rejection: %v", err)
			}
			if stored.MustChangePassword != tc.mustChange {
				t.Errorf("must_change_password = %v, want unchanged %v", stored.MustChangePassword, tc.mustChange)
			}
		})
	}
}

// TestChangePasswordRejectsUnknownAuthRequest 伪造/过期的授权票据一律按会话不存在处理。
func TestChangePasswordRejectsUnknownAuthRequest(t *testing.T) {
	provider, _ := newChangePasswordEnv(t)

	svc := &oidcAuthSvc{provider: provider}
	err := svc.ChangePassword(newOIDCGinCtx(t, "/oidc/login/changePassword"), &dtooidc.OIDCChangePasswordReq{
		AuthRequestID:   "not-exist",
		CurrentPassword: "Temp1234",
		NewPassword:     "Fresh1234",
	})
	if err == nil || err.Error() != code.GetError(code.OIDCSessionNotFound).Error() {
		t.Fatalf("ChangePassword err = %v, want %v", err, code.GetError(code.OIDCSessionNotFound))
	}
}
