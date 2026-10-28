package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleCond struct {
	*gormdao.BaseCond
	TenantID   string
	AppID      string
	IDs        []string
	Name       string
	Code       model.RoleCode // 精确匹配角色编码（跨系统授权契约值，(租户, 应用) 内唯一）
	Source     model.RoleSource
	AdminType  model.SysAdminType // 精确匹配系统管理类型(admin/normal)
	Keyword    string             // 模糊搜索: 名称/编码 LIKE
	Unassigned bool               // 仅未归属应用的角色(app_id 为空串)
}

func (c *RoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	// app_id 为空串是"未归属应用"（系统角色）的存储值，而 AppID 空值语义是"不过滤"，故用独立开关表达
	if c.Unassigned {
		db.Where(clause.Eq{Column: col("app_id"), Value: ""})
	}
	if len(c.IDs) > 0 {
		db.Where(clause.IN{Column: col("id"), Values: toAnySlice(c.IDs)})
	}
	if c.Keyword != "" {
		db.Where(buildOrLike(col, c.Keyword, "name", "code"))
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Code != "" {
		db.Where(clause.Eq{Column: col("code"), Value: c.Code})
	}
	if c.Source != "" {
		db.Where(clause.Eq{Column: col("source"), Value: c.Source})
	}
	if c.AdminType != "" {
		db.Where(clause.Eq{Column: col("admin_type"), Value: c.AdminType})
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
