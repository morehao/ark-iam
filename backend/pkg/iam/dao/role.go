package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleCond struct {
	*gormdao.BaseCond
	TenantID uint
	AppID    uint
	Name     string
	Code     string
	Type     string
}

func (c *RoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.AppID != 0 {
		db.Where(tableName+".app_id = ?", c.AppID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.Type != "" {
		db.Where(tableName+".type = ?", c.Type)
	}
}

type RoleDao struct {
	*gormdao.Dao[model.RoleEntity, model.RoleEntityList]
}

func NewRoleDao() *RoleDao {
	return &RoleDao{
		Dao: gormdao.NewDao[model.RoleEntity, model.RoleEntityList](
			model.TableNameRole, "RoleDao",
			dbclient.IamDB, gormdao.WithoutSoftDelete(),
		),
	}
}
