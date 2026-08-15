package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type AuditLogCond struct {
	*gormdao.BaseCond
	PersonID uint
	TenantID uint
	Action   string
	Result   string
}

func (c *AuditLogCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != 0 {
		db.Where(tableName+".actor_person_id = ?", c.PersonID)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Action != "" {
		db.Where(tableName+".action = ?", c.Action)
	}
	if c.Result != "" {
		db.Where(tableName+".result = ?", c.Result)
	}
}

type AuditLogDao struct {
	*gormdao.Dao[model.AuditLogEntity, model.AuditLogEntityList, uint]
}

func NewAuditLogDao(opts ...DaoOption) *AuditLogDao {
	return &AuditLogDao{
		Dao: gormdao.NewDao[model.AuditLogEntity, model.AuditLogEntityList, uint](model.TableNameAuditLog, "AuditLogDao", resolveDBGetter(opts...)),
	}
}
