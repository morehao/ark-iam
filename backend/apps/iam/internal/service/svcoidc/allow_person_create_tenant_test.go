package svcoidc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveAllowPersonCreateTenant(t *testing.T) {
	cases := []struct {
		name        string
		clientID    string
		tenantCount int
		client      *model.OAuthClientEntity
		app         *model.ApplicationEntity
		want        bool
	}{
		{
			name:        "person already has tenants => false even if policy allows",
			clientID:    "cid-1",
			tenantCount: 1,
			client:      &model.OAuthClientEntity{AppID: 7},
			app:         &model.ApplicationEntity{TenantPolicy: datatypes.JSON(`{"allowPersonCreateTenant":true}`)},
			want:        false,
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
			client:      &model.OAuthClientEntity{AppID: 7},
			app:         &model.ApplicationEntity{TenantPolicy: datatypes.JSON(`{"allowPersonCreateTenant":true}`)},
			want:        true,
		},
		{
			name:        "policy disallow => false",
			clientID:    "cid-3",
			tenantCount: 0,
			client:      &model.OAuthClientEntity{AppID: 8},
			app:         &model.ApplicationEntity{TenantPolicy: datatypes.JSON(`{"allowPersonCreateTenant":false}`)},
			want:        false,
		},
		{
			name:        "policy absent => false",
			clientID:    "cid-4",
			tenantCount: 0,
			client:      &model.OAuthClientEntity{AppID: 9},
			app:         &model.ApplicationEntity{TenantPolicy: datatypes.JSON(`{}`)},
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
			db := newAllowPersonCreateTenantTestDB(t)
			if c.client != nil && c.clientID != "" {
				client := c.client
				client.Model = gorm.Model{ID: client.AppID}
				client.ClientID = c.clientID
				client.RedirectURIs = datatypes.JSON(`[]`)
				client.PostLogoutRedirectURIs = datatypes.JSON(`[]`)
				client.GrantTypes = datatypes.JSON(`["authorization_code"]`)
				client.ResponseTypes = datatypes.JSON(`["code"]`)
				client.AllowedOrigins = datatypes.JSON(`[]`)
				client.DefaultScopes = datatypes.JSON(`["openid"]`)
				if err := db.Create(client).Error; err != nil {
					t.Fatalf("seed client: %v", err)
				}
			}
			if c.client != nil && c.app != nil && c.clientID != "" {
				app := c.app
				app.Model = gorm.Model{ID: c.client.AppID}
				app.Code = fmt.Sprintf("app-%d", c.client.AppID)
				if err := db.Create(app).Error; err != nil {
					t.Fatalf("seed app: %v", err)
				}
			}

			svc := &oidcAuthSvc{
				oauthClientDao: func() *dao.OAuthClientDao {
					return dao.NewOAuthClientDaoWithDB(dbGetter(db))
				},
				applicationDao: func() *dao.ApplicationDao {
					return dao.NewApplicationDaoWithDB(dbGetter(db))
				},
			}

			ginCtx, _ := gin.CreateTestContext(nil)
			got := svc.resolveAllowPersonCreateTenant(ginCtx, c.clientID, c.tenantCount)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func dbGetter(db *gorm.DB) func(context.Context) *gorm.DB {
	return func(c context.Context) *gorm.DB { return db.WithContext(c) }
}

func newAllowPersonCreateTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:allow_person_create_tenant_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.OAuthClientEntity{}, &model.ApplicationEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
