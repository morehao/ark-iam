package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type RoleCond struct {
	*genericdao.BaseCond
	TenantID uint
	Name     string
	Code     string
	Type     string
}

func (c *RoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
	if c.Code != "" {
		db.Where(tableName + ".code = ?", c.Code)
	}
	if c.Type != "" {
		db.Where(tableName + ".type = ?", c.Type)
	}
}

type RoleDao struct {
	*genericdao.GenericDao[model.RoleEntity, model.RoleEntityList]
}

func NewRoleDao() *RoleDao {
	return &RoleDao{
		GenericDao: genericdao.NewGenericDao[model.RoleEntity, model.RoleEntityList](
			model.TableNameRole, "RoleDao",
			dbclient.IamDB,
		),
	}
}