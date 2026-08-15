package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationUserCond struct {
	*gormdao.BaseCond
	TenantID       uint
	OrganizationID uint
	UserID         uint
}

func (c *OrganizationUserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.UserID != 0 {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
}

type OrganizationUserDao struct {
	*gormdao.Dao[model.OrganizationUserEntity, model.OrganizationUserEntityList]
}

func NewOrganizationUserDao(opts ...DaoOption) *OrganizationUserDao {
	return &OrganizationUserDao{
		Dao: gormdao.NewDao[model.OrganizationUserEntity, model.OrganizationUserEntityList](
			model.TableNameOrganizationUser, "OrganizationUserDao",
			resolveDBGetter(opts...),
		),
	}
}
