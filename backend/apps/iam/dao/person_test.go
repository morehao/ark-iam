package dao

import (
	"testing"

	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPersonCondBuildConditionSupportsGlobalIdentifiers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonEntity{}); err != nil {
		t.Fatalf("migrate person: %v", err)
	}
	seed := []model.PersonEntity{
		{Username: "alice", PrimaryEmail: "alice@example.com", PrimaryPhone: "13800000001", Name: "Alice", Profile: []byte(`{}`), CustomData: []byte(`{}`)},
		{Username: "bob", PrimaryEmail: "bob@example.com", PrimaryPhone: "13800000002", Name: "Bob", Profile: []byte(`{}`), CustomData: []byte(`{}`)},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed persons: %v", err)
	}

	query := db.Model(&model.PersonEntity{}).Table(model.TableNamePerson)
	cond := &PersonCond{Username: "alice", PrimaryEmail: "alice@example.com"}
	cond.BuildCondition(query, model.TableNamePerson)

	var list model.PersonEntityList
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("find persons: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(list))
	}
	if list[0].Username != "alice" || list[0].PrimaryPhone != "13800000001" {
		t.Fatalf("unexpected filtered row: %+v", list[0])
	}

	phoneQuery := db.Model(&model.PersonEntity{}).Table(model.TableNamePerson)
	phoneCond := &PersonCond{PrimaryPhone: "13800000002"}
	phoneCond.BuildCondition(phoneQuery, model.TableNamePerson)

	var phoneList model.PersonEntityList
	if err := phoneQuery.Find(&phoneList).Error; err != nil {
		t.Fatalf("find persons by phone: %v", err)
	}
	if len(phoneList) != 1 || phoneList[0].Username != "bob" {
		t.Fatalf("unexpected phone filtered rows: %+v", phoneList)
	}
}
