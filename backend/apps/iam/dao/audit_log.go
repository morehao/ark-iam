package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
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
	*gormdao.Dao[model.AuditLogEntity, model.AuditLogEntityList]
	dbGetter gormdao.DBGetter
}

func NewAuditLogDao() *AuditLogDao {
	return &AuditLogDao{
		Dao:      gormdao.NewDao[model.AuditLogEntity, model.AuditLogEntityList](model.TableNameAuditLog, "AuditLogDao", dbclient.IamDB),
		dbGetter: dbclient.IamDB,
	}
}

func NewAuditLogDaoWithDB(dbGetter gormdao.DBGetter) *AuditLogDao {
	return &AuditLogDao{
		Dao:      gormdao.NewDao[model.AuditLogEntity, model.AuditLogEntityList](model.TableNameAuditLog, "AuditLogDao", dbGetter),
		dbGetter: dbGetter,
	}
}
