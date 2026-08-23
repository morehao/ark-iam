package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleCond struct {
	*gormdao.BaseCond
	TenantID          string
	AppID             string
	IDs               []string
	Name              string
	Code              string
	Source            string
	AdminLevel        string // 精确匹配系统管理等级
	AdminLevelAtLeast string // 按门槛匹配：admin_level 等级 >= 该值（如 "basic" 命中 basic/super）
	Keyword           string // 模糊搜索: 名称/编码 LIKE
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
		db.Where(tableName+".name LIKE ? OR "+tableName+".code LIKE ?", k, k)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.Source != "" {
		db.Where(tableName+".source = ?", c.Source)
	}
	if c.AdminLevel != "" {
		db.Where(tableName+".admin_level = ?", c.AdminLevel)
	}
	if c.AdminLevelAtLeast != "" {
		db.Where(tableName+".admin_level IN ?", adminLevelsAtLeast(c.AdminLevelAtLeast))
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

// adminLevelsAtLeast 返回等级 >= threshold 的全部等级取值（member < super）。
func adminLevelsAtLeast(threshold string) []string {
	switch model.SysAdminLevel(threshold).SysAdminRank() {
	case model.SysAdminLevelSuper.SysAdminRank():
		return []string{string(model.SysAdminLevelSuper)}
	default:
		return []string{string(model.SysAdminLevelMember), string(model.SysAdminLevelSuper)}
	}
}
