package dao

import (
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserCondBuildConditionAppliesIsSuspendedZeroFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	now := time.Now()
	seed := []model.UserEntity{
		{TenantID: 1, PersonID: 11, Name: "active-user", Profile: []byte(`{}`), CustomData: []byte(`{}`), IsSuspended: 0, IsOwner: 0, JoinedAt: &now},
		{TenantID: 1, PersonID: 22, Name: "suspended-user", Profile: []byte(`{}`), CustomData: []byte(`{}`), IsSuspended: 1, IsOwner: 0, JoinedAt: &now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	isSuspended := int8(0)
	query := db.Model(&model.UserEntity{}).Table(model.TableNameUser)
	cond := &UserCond{TenantID: 1, IsSuspended: &isSuspended}
	cond.BuildCondition(query, model.TableNameUser)

	var list model.UserEntityList
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("find users: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(list))
	}
	if list[0].Name != "active-user" || list[0].IsSuspended != 0 {
		t.Fatalf("unexpected filtered row: name=%q is_suspended=%d", list[0].Name, list[0].IsSuspended)
	}
}
