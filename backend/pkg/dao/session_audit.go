package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionAuditCond struct {
	*gormdao.BaseCond
	PersonID  string
	SessionID string
}

func (c *SessionAuditCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.PersonID != "" {
		db.Where(clause.Eq{Column: col("person_id"), Value: c.PersonID})
	}
	if c.SessionID != "" {
		db.Where(clause.Eq{Column: col("session_id"), Value: c.SessionID})
	}
}

type SessionAuditDao struct {
	*gormdao.Dao[model.SessionAuditEntity, model.SessionAuditEntityList, string]
}

func NewSessionAuditDao(opts ...DaoOption) *SessionAuditDao {
	return &SessionAuditDao{
		Dao: gormdao.NewDao[model.SessionAuditEntity, model.SessionAuditEntityList, string](
			model.TableNameSession, "SessionAuditDao",
			resolveDBGetter(opts...),
		),
	}
}
