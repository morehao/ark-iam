package dtooidc

import (
	"encoding/json"
	"testing"

	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
)

func TestRegisterPersonRespJSON(t *testing.T) {
	resp := RegisterPersonResp{
		PersonID:                "p-1",
		RequiresTenantSelection: true,
		Tenants:                 []objauth.TenantOption{{TenantID: "t-1", Name: "租户A"}},
		AllowPersonCreateTenant: true,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"personID":"p-1","requiresTenantSelection":true,"tenants":[{"tenantID":"t-1","name":"租户A","tag":"","userID":"","isOwner":false}],"allowPersonCreateTenant":true}`
	if string(b) != want {
		t.Fatalf("\ngot  %s\nwant %s", b, want)
	}
}

func TestOIDCLoginRespPersonIDOmitEmpty(t *testing.T) {
	b, err := json.Marshal(OIDCLoginResp{ContinueURL: "u", AllowPersonCreateTenant: false})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `{"continueURL":"u","allowPersonCreateTenant":false}` {
		t.Fatalf("personID should be omitempty, got: %s", b)
	}
}
