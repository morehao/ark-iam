package dao

import (
	"testing"

	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserDepartmentRelationCondBuildConditionAppliesIsPrimaryZeroFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserDepartmentRelationEntity{}); err != nil {
		t.Fatalf("migrate user department relation: %v", err)
	}
	seed := []model.UserDepartmentRelationEntity{
		{TenantID: 1, UserID: 11, DepartmentID: 101, IsPrimary: 0},
		{TenantID: 1, UserID: 11, DepartmentID: 102, IsPrimary: 1},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed user department relations: %v", err)
	}

	isPrimary := int8(0)
	query := db.Model(&model.UserDepartmentRelationEntity{}).Table(model.TableNameUserDepartmentRelation)
	cond := &UserDepartmentRelationCond{TenantID: 1, UserID: 11, IsPrimary: &isPrimary}
	cond.BuildCondition(query, model.TableNameUserDepartmentRelation)

	var list model.UserDepartmentRelationEntityList
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("find user department relations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(list))
	}
	if list[0].DepartmentID != 101 || list[0].IsPrimary != 0 {
		t.Fatalf("unexpected filtered row: department_id=%d is_primary=%d", list[0].DepartmentID, list[0].IsPrimary)
	}
}
