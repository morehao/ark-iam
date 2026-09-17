package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleCond struct {
	*gormdao.BaseCond
	TenantID   string
	AppID      string
	IDs        []string
	Name       string
	Code       model.RoleCode // 精确匹配角色编码（跨系统授权契约值，租户内唯一）
	Source     model.RoleSource
	AdminType  model.SysAdminType // 精确匹配系统管理类型(admin/normal)
	Keyword    string             // 模糊搜索: 名称/编码 LIKE
	Unassigned bool               // 仅未归属应用的角色(app_id 为空串)
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
	// app_id 为空串是"未归属应用"（系统角色）的存储值，而 AppID 空值语义是"不过滤"，故用独立开关表达
	if c.Unassigned {
		db.Where(tableName + ".app_id = ''")
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
