package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuditLogCond struct {
	*gormdao.BaseCond
	PersonID string
	TenantID string
	Action   model.AuditAction
	Result   model.AuditResult
}

func (c *AuditLogCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.PersonID != "" {
		db.Where(clause.Eq{Column: col("actor_person_id"), Value: c.PersonID})
	}
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.Action != "" {
		db.Where(clause.Eq{Column: col("action"), Value: c.Action})
	}
	if c.Result != "" {
		db.Where(clause.Eq{Column: col("result"), Value: c.Result})
	}
}

type AuditLogDao struct {
	*gormdao.Dao[model.AuditLogEntity, model.AuditLogEntityList, string]
}

func NewAuditLogDao(opts ...DaoOption) *AuditLogDao {
	return &AuditLogDao{
		Dao: gormdao.NewDao[model.AuditLogEntity, model.AuditLogEntityList, string](model.TableNameAuditLog, "AuditLogDao", resolveDBGetter(opts...)),
	}
}
