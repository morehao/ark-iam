package model

import (
	"reflect"
	"testing"
	"time"
)

func TestConnectorEntityTableName(t *testing.T) {
	if (ConnectorEntity{}).TableName() != TableNameConnector {
		t.Fatalf("unexpected table name: %s", (ConnectorEntity{}).TableName())
	}
}

func TestUserIdentityEntityTableName(t *testing.T) {
	if (UserIdentityEntity{}).TableName() != TableNameUserIdentity {
		t.Fatalf("unexpected table name: %s", (UserIdentityEntity{}).TableName())
	}
}

func TestPersonEntityTableName(t *testing.T) {
	if got := (PersonEntity{}).TableName(); got != TableNamePerson {
		t.Fatalf("unexpected table name: %s", got)
	}
}

func TestConnectorAndUserIdentityModelDoNotKeepLegacyCompatFields(t *testing.T) {
	connectorType := reflect.TypeOf(ConnectorEntity{})
	for _, fieldName := range []string{"ConnectorID", "Metadata"} {
		if _, ok := connectorType.FieldByName(fieldName); ok {
			t.Fatalf("connector model still has legacy field: %s", fieldName)
		}
	}

	userIdentityType := reflect.TypeOf(UserIdentityEntity{})
	if _, ok := userIdentityType.FieldByName("IdentityID"); ok {
		t.Fatalf("user identity model still has legacy field: IdentityID")
	}
}

func TestUserModelUsesPersonCenteredFields(t *testing.T) {
	userType := reflect.TypeOf(UserEntity{})
	for _, fieldName := range []string{"PersonID", "IsOwner", "JoinedAt"} {
		if _, ok := userType.FieldByName(fieldName); !ok {
			t.Fatalf("user model missing field: %s", fieldName)
		}
	}
	for _, fieldName := range []string{"Username", "PrimaryEmail", "PrimaryPhone", "PasswordEncrypted", "PasswordMethod", "AppID", "Identities"} {
		if _, ok := userType.FieldByName(fieldName); ok {
			t.Fatalf("user model still has legacy field: %s", fieldName)
		}
	}
}

func TestUserIdentityModelUsesPersonCenteredFields(t *testing.T) {
	identityType := reflect.TypeOf(UserIdentityEntity{})
	for _, fieldName := range []string{"PersonID", "Provider", "LastUsedAt"} {
		if _, ok := identityType.FieldByName(fieldName); !ok {
			t.Fatalf("user identity model missing field: %s", fieldName)
		}
	}
	for _, fieldName := range []string{"TenantID", "UserID"} {
		if _, ok := identityType.FieldByName(fieldName); ok {
			t.Fatalf("user identity model still has legacy field: %s", fieldName)
		}
	}
}

func TestRefreshTokenModelUsesPersonCenteredFields(t *testing.T) {
	refreshTokenType := reflect.TypeOf(RefreshTokenEntity{})
	for _, fieldName := range []string{"PersonID", "SessionID", "ClientType", "ClientIP", "UserAgent"} {
		if _, ok := refreshTokenType.FieldByName(fieldName); !ok {
			t.Fatalf("refresh token model missing field: %s", fieldName)
		}
	}
}

func TestUserLoginLogModelUsesPersonCenteredFields(t *testing.T) {
	loginLogType := reflect.TypeOf(UserLoginLogEntity{})
	for _, fieldName := range []string{"PersonID", "LoginType"} {
		if _, ok := loginLogType.FieldByName(fieldName); !ok {
			t.Fatalf("user login log model missing field: %s", fieldName)
		}
	}
}

func TestBusinessTimeFieldsUseTimePointerInsteadOfDeletedAt(t *testing.T) {
	timePtrType := reflect.TypeOf((*time.Time)(nil))
	checks := []struct {
		name      string
		modelType reflect.Type
		fieldName string
	}{
		{name: "person last sign in", modelType: reflect.TypeOf(PersonEntity{}), fieldName: "LastSignInAt"},
		{name: "user joined at", modelType: reflect.TypeOf(UserEntity{}), fieldName: "JoinedAt"},
		{name: "user last sign in", modelType: reflect.TypeOf(UserEntity{}), fieldName: "LastSignInAt"},
		{name: "user identity last used", modelType: reflect.TypeOf(UserIdentityEntity{}), fieldName: "LastUsedAt"},
		{name: "refresh token expires at", modelType: reflect.TypeOf(RefreshTokenEntity{}), fieldName: "ExpiredAt"},
		{name: "refresh token revoked at", modelType: reflect.TypeOf(RefreshTokenEntity{}), fieldName: "RevokedAt"},
	}

	for _, check := range checks {
		field, ok := check.modelType.FieldByName(check.fieldName)
		if !ok {
			t.Fatalf("%s missing field %s", check.name, check.fieldName)
		}
		if field.Type != timePtrType {
			t.Fatalf("%s expected %s to use *time.Time, got %s", check.name, check.fieldName, field.Type)
		}
	}
}
