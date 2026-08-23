package svcauth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
)

func zeroTenantLoginCtx(t *testing.T) *gin.Context {
	t.Helper()
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = httptestRequest(t)
	return ginCtx
}

func seedZeroTenantPerson(t *testing.T, identifier, password string) {
	t.Helper()
	hash, err := gcrypto.GeneratePasswordHash(password)
	if err != nil {
		t.Fatalf("GeneratePasswordHash: %v", err)
	}
	person := &model.PersonEntity{
		Username:          model.StrPtr(identifier),
		PasswordEncrypted: hash,
		PasswordMethod:    "bcrypt",
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
	}
	if strings.Contains(identifier, "@") {
		person.Username = nil
		person.PrimaryEmail = model.StrPtr(identifier)
	} else if len(identifier) >= 11 && strings.HasPrefix(identifier, "1") {
		person.Username = nil
		person.PrimaryPhone = model.StrPtr(identifier)
	}
	if err := testutil.SetupSQLite(t, &model.PersonEntity{}, &model.UserEntity{}, &model.TenantEntity{}).Create(person).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
}

func TestAuthenticatePasswordZeroTenantPersonCorrectPassword(t *testing.T) {
	seedZeroTenantPerson(t, "zero@example.com", "Password1")

	person, user, tenants, err := NewAuthSvc().AuthenticatePassword(zeroTenantLoginCtx(t), "zero@example.com", "Password1")
	if err != nil {
		t.Fatalf("AuthenticatePassword returned error: %v", err)
	}
	if person == nil || person.ID == "" {
		t.Fatalf("expected resolved person, got %#v", person)
	}
	if user != nil {
		t.Fatalf("expected nil user for zero-tenant person, got %#v", user)
	}
	if len(tenants) != 0 {
		t.Fatalf("expected empty tenants for zero-tenant person, got %#v", tenants)
	}
}

func TestAuthenticatePasswordZeroTenantPersonWrongPassword(t *testing.T) {
	seedZeroTenantPerson(t, "zero@example.com", "Password1")

	person, user, tenants, err := NewAuthSvc().AuthenticatePassword(zeroTenantLoginCtx(t), "zero@example.com", "wrong-password")
	assertCode(t, err, code.AuthLoginFailedError)
	if person != nil || user != nil || len(tenants) != 0 {
		t.Fatalf("expected all nil/empty on wrong password, got person:%#v user:%#v tenants:%#v", person, user, tenants)
	}
}

func TestAuthenticatePasswordPersonNotExist(t *testing.T) {
	testutil.SetupSQLite(t, &model.PersonEntity{}, &model.UserEntity{}, &model.TenantEntity{})

	person, user, tenants, err := NewAuthSvc().AuthenticatePassword(zeroTenantLoginCtx(t), "ghost@example.com", "whatever")
	assertCode(t, err, code.AuthLoginFailedError)
	if person != nil || user != nil || len(tenants) != 0 {
		t.Fatalf("expected all nil/empty when person not found, got person:%#v user:%#v tenants:%#v", person, user, tenants)
	}
}

func TestAuthenticatePasswordZeroTenantPersonVariableIdentifierForms(t *testing.T) {
	for _, tc := range []struct {
		identifier string
	}{
		{"zero-username"},
		{"zero@example.com"},
		{"13800000000"},
	} {
		t.Run(tc.identifier, func(t *testing.T) {
			var username, email, phone *string
			switch {
			case tc.identifier == "zero-username":
				username = model.StrPtr(tc.identifier)
			case len(tc.identifier) >= 11 && tc.identifier[0] == '1':
				phone = model.StrPtr(tc.identifier)
			default:
				email = model.StrPtr(tc.identifier)
			}
			hash, err := gcrypto.GeneratePasswordHash("Password1")
			if err != nil {
				t.Fatalf("hash: %v", err)
			}
			dbt := testutil.SetupSQLite(t, &model.PersonEntity{}, &model.UserEntity{}, &model.TenantEntity{})
			if err := dbt.Create(&model.PersonEntity{
				Username:          username,
				PrimaryEmail:      email,
				PrimaryPhone:      phone,
				PasswordEncrypted: hash,
				PasswordMethod:    "bcrypt",
				Profile:           json.RawMessage(`{}`),
				CustomData:        json.RawMessage(`{}`),
			}).Error; err != nil {
				t.Fatalf("seed person: %v", err)
			}
			person, user, tenants, err := NewAuthSvc().AuthenticatePassword(zeroTenantLoginCtx(t), tc.identifier, "Password1")
			if err != nil {
				t.Fatalf("AuthenticatePassword returned error: %v", err)
			}
			if person == nil || person.ID == "" {
				t.Fatalf("expected resolved person, got %#v", person)
			}
			if user != nil || len(tenants) != 0 {
				t.Fatalf("expected nil user + empty tenants for zero-tenant, got user:%#v tenants:%#v", user, tenants)
			}
		})
	}
}

func TestAuthenticatePasswordPersonWithTenantPreservesOriginalBehavior(t *testing.T) {
	hash, err := gcrypto.GeneratePasswordHash("Password1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	now := time.Now()
	dbt := testutil.SetupSQLite(t, &model.PersonEntity{}, &model.UserEntity{}, &model.TenantEntity{})
	if err := dbt.Create(&model.PersonEntity{
		BaseEntity:        gormdao.BaseEntity{StringID: gormdao.StringID{ID: "p1"}},
		PrimaryEmail:      model.StrPtr("member@example.com"),
		PasswordEncrypted: hash,
		PasswordMethod:    "bcrypt",
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
	}).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	if err := dbt.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "t1"}},
		Code:       "t1",
		Name:       "tenant-1",
	}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := dbt.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "u1"}},
		TenantID:   "t1",
		PersonID:   "p1",
		Name:       "member",
		Profile:    json.RawMessage(`{}`),
		CustomData: json.RawMessage(`{}`),
		JoinedAt:   &now,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	person, user, tenants, err := NewAuthSvc().AuthenticatePassword(zeroTenantLoginCtx(t), "member@example.com", "Password1")
	if err != nil {
		t.Fatalf("AuthenticatePassword returned error: %v", err)
	}
	if person == nil || person.ID != "p1" {
		t.Fatalf("expected person p1, got %#v", person)
	}
	if user == nil || user.ID != "u1" || user.TenantID != "t1" {
		t.Fatalf("expected member user u1, got %#v", user)
	}
	if len(tenants) != 1 || tenants[0].TenantID != "t1" {
		t.Fatalf("expected tenant t1, got %#v", tenants)
	}
}
