package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type SessionAuditCond struct {
	*gormdao.BaseCond
	PersonID  uint
	SessionID string
	Status    string
}

func (c *SessionAuditCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != 0 {
		db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.SessionID != "" {
		db.Where(tableName+".session_id = ?", c.SessionID)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
}

type SessionAuditDao struct {
	*gormdao.Dao[model.SessionAuditEntity, model.SessionAuditEntityList]
	dbGetter gormdao.DBGetter
}

func NewSessionAuditDao() *SessionAuditDao {
	return &SessionAuditDao{
		Dao: gormdao.NewDao[model.SessionAuditEntity, model.SessionAuditEntityList](
			model.TableNameSession, "SessionAuditDao",
			dbclient.IamDB, gormdao.WithoutSoftDelete(),
		),
		dbGetter: dbclient.IamDB,
	}
}

func NewSessionAuditDaoWithDB(dbGetter gormdao.DBGetter) *SessionAuditDao {
	return &SessionAuditDao{
		Dao: gormdao.NewDao[model.SessionAuditEntity, model.SessionAuditEntityList](
			model.TableNameSession, "SessionAuditDao",
			dbGetter, gormdao.WithoutSoftDelete(),
		),
		dbGetter: dbGetter,
	}
}
