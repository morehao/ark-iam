package model

import (
	"reflect"
	"testing"
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
