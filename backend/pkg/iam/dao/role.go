package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleCond struct {
	*gormdao.BaseCond
	TenantID  string
	AppID     string
	IDs       []string
	Name      string
	Source    string
	AdminType model.SysAdminType // 精确匹配系统管理类型(admin/normal)
	Keyword   string             // 模糊搜索: 名称 LIKE
}

func (c *RoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.AppID != "" {
		db.Where(tableName+".app_id = ?", c.AppID)
	}
	if len(c.IDs) > 0 {
		db.Where(tableName+".id IN ?", c.IDs)
	}
	if c.Keyword != "" {
		k := "%" + c.Keyword + "%"
		db.Where(tableName+".name LIKE ?", k)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Source != "" {
		db.Where(tableName+".source = ?", c.Source)
	}
	if c.AdminType != "" {
		db.Where(tableName+".admin_type = ?", c.AdminType)
	}
}

type RoleDao struct {
	*gormdao.Dao[model.RoleEntity, model.RoleEntityList, string]
}

func NewRoleDao(opts ...DaoOption) *RoleDao {
	return &RoleDao{
		Dao: gormdao.NewDao[model.RoleEntity, model.RoleEntityList, string](
			model.TableNameRole, "RoleDao",
			resolveDBGetter(opts...),
		),
	}
}
