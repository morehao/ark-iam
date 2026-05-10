package dao

import (
	"testing"

	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserIdentityCondBuildConditionAppliesUserAndIssuerFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserIdentityEntity{}); err != nil {
		t.Fatalf("migrate user identity: %v", err)
	}
	seed := []model.UserIdentityEntity{
		{TenantID: 1, UserID: 11, Issuer: "issuer-a", ExternalSubject: "external-a", Detail: []byte(`{}`)},
		{TenantID: 1, UserID: 22, Issuer: "issuer-a", ExternalSubject: "external-b", Detail: []byte(`{}`)},
		{TenantID: 1, UserID: 11, Issuer: "issuer-b", ExternalSubject: "external-c", Detail: []byte(`{}`)},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed user identities: %v", err)
	}

	query := db.Model(&model.UserIdentityEntity{}).Table(model.TableNameUserIdentity)
	cond := &UserIdentityCond{TenantID: 1, UserID: 11, Issuer: "issuer-a"}
	cond.BuildCondition(query, model.TableNameUserIdentity)

	var list model.UserIdentityEntityList
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("find user identities: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(list))
	}
	if list[0].UserID != 11 || list[0].Issuer != "issuer-a" {
		t.Fatalf("unexpected filtered row: user_id=%d issuer=%q", list[0].UserID, list[0].Issuer)
	}
}
