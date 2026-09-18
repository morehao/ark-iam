package svcoidc

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/testutil"
	"github.com/morehao/ark-iam/pkg/model"
)

func TestResolveAllowPersonCreateTenant(t *testing.T) {
	cases := []struct {
		name        string
		clientID    string
		tenantCount int
		client      *model.ApplicationClientEntity
		app         *model.ApplicationEntity
		want        bool
	}{
		{
			name:        "person already has tenants => false even if policy allows",
			clientID:    "cid-1",
			tenantCount: 1,
			client:      &model.ApplicationClientEntity{AppID: "7"},
			app: &model.ApplicationEntity{
				AllowPersonCreateTenant: model.AppPersonCreateTenantPolicyEnable,
			},
			want: false,
		},
		{
			name:        "empty client id => false",
			clientID:    "",
			tenantCount: 0,
			want:        false,
		},
		{
			name:        "policy allow => true",
			clientID:    "cid-2",
			tenantCount: 0,
			client:      &model.ApplicationClientEntity{AppID: "7"},
			app: &model.ApplicationEntity{
				AllowPersonCreateTenant: model.AppPersonCreateTenantPolicyEnable,
			},
			want: true,
		},
		{
			name:        "policy disallow => false",
			clientID:    "cid-3",
			tenantCount: 0,
			client:      &model.ApplicationClientEntity{AppID: "8"},
			app: &model.ApplicationEntity{
				AllowPersonCreateTenant: model.AppPersonCreateTenantPolicyDisable,
			},
			want: false,
		},
		{
			name:        "policy absent => false",
			clientID:    "cid-4",
			tenantCount: 0,
			client:      &model.ApplicationClientEntity{AppID: "9"},
			app:         &model.ApplicationEntity{},
			want:        false,
		},
		{
			name:        "no matching client => false",
			clientID:    "cid-missing",
			tenantCount: 0,
			want:        false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationEntity{})
			if c.client != nil && c.clientID != "" {
				client := c.client
				client.ID = client.AppID
				client.Code = c.clientID
				client.RedirectURIs = model.RedirectURIList{}
				client.PostLogoutRedirectURIs = model.PostLogoutRedirectURIList{}
				client.GrantTypes = model.GrantTypeList{model.GrantTypeAuthorizationCode}
				client.ResponseTypes = model.ResponseTypeList{model.ResponseTypeCode}
				client.AllowedOrigins = model.AllowedOriginList{}
				client.DefaultScopes = model.DefaultScopeList{model.ScopeOpenID}
				if err := db.Create(client).Error; err != nil {
					t.Fatalf("seed client: %v", err)
				}
			}
			if c.client != nil && c.app != nil && c.clientID != "" {
				app := c.app
				app.ID = c.client.AppID
				app.Code = fmt.Sprintf("app-%s", c.client.AppID)
				if err := db.Create(app).Error; err != nil {
					t.Fatalf("seed app: %v", err)
				}
			}
			svc := &oidcAuthSvc{}

			ginCtx, _ := gin.CreateTestContext(nil)
			got := svc.resolveAllowPersonCreateTenant(ginCtx, c.clientID, c.tenantCount)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func TestAppAllowsPersonCreateTenant(t *testing.T) {
	cases := []struct {
		name  string
		allow model.AppPersonCreateTenantPolicy
		want  bool
	}{
		{name: "enable", allow: model.AppPersonCreateTenantPolicyEnable, want: true},
		{name: "disable", allow: model.AppPersonCreateTenantPolicyDisable, want: false},
		{name: "absent", allow: "", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationEntity{})
			app := &model.ApplicationEntity{Code: "app-x", AllowPersonCreateTenant: c.allow}
			if err := db.Create(app).Error; err != nil {
				t.Fatalf("seed app: %v", err)
			}
			client := &model.ApplicationClientEntity{Code: "cid-x", AppID: app.ID}
			client.ID = client.AppID
			client.RedirectURIs = model.RedirectURIList{}
			client.PostLogoutRedirectURIs = model.PostLogoutRedirectURIList{}
			client.GrantTypes = model.GrantTypeList{model.GrantTypeAuthorizationCode}
			client.ResponseTypes = model.ResponseTypeList{model.ResponseTypeCode}
			client.AllowedOrigins = model.AllowedOriginList{}
			client.DefaultScopes = model.DefaultScopeList{model.ScopeOpenID}
			if err := db.Create(client).Error; err != nil {
				t.Fatalf("seed client: %v", err)
			}
			svc := &oidcAuthSvc{}
			ginCtx, _ := gin.CreateTestContext(nil)
			if got := svc.appAllowsPersonCreateTenant(ginCtx, "cid-x"); got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
