package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MenuOrderBySort 菜单的规范排序字段：先按控制台可维护的「排序」(sort) 升序，
// 再按 code 升序兜底（同应用内 code 唯一，保证顺序确定）。
// 菜单树按 parent_id 分组构建，该排序等价于「同一父级下按 sort 升序」。
// 菜单顺序是运维字段（种子 create_only，控制台可改），因此所有面向界面的菜单查询都必须显式带上它——
// 缺省时数据库不保证任何顺序，控制台里改「排序」将完全看不到效果。
const MenuOrderBySort = "sort, code"

type MenuCond struct {
	*gormdao.BaseCond
	AppID      string
	ParentID   string
	Name       string
	Code       string
	SeedKey    string // 种子身份键：内置菜单的稳定标识（定位内置菜单请用它，而不是可被改名的 Code）
	Type       model.MenuType
	Status     model.MenuStatus
	Visibility model.MenuVisibility
}

func (c *MenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	if c.ParentID != "" {
		db.Where(clause.Eq{Column: col("parent_id"), Value: c.ParentID})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Code != "" {
		db.Where(clause.Eq{Column: col("code"), Value: c.Code})
	}
	if c.SeedKey != "" {
		db.Where(clause.Eq{Column: col("seed_key"), Value: c.SeedKey})
	}
	if c.Type != "" {
		db.Where(clause.Eq{Column: col("type"), Value: c.Type})
	}
	if c.Visibility != "" {
		db.Where(clause.Eq{Column: col("visibility"), Value: c.Visibility})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
}

type MenuDao struct {
	*gormdao.Dao[model.MenuEntity, model.MenuEntityList, string]
}

func NewMenuDao(opts ...DaoOption) *MenuDao {
	return &MenuDao{
		Dao: gormdao.NewDao[model.MenuEntity, model.MenuEntityList, string](
			model.TableNameMenu, "MenuDao",
			resolveDBGetter(opts...),
		),
	}
}

// MenuScopedCond 带 app 约束的菜单查询（租户侧安全写路径用）：
// menu 是全局表（无 tenant_id），租户侧写必须显式带 app_id 兜底，确保只作用于"已确认归属
// 当前租户"的应用菜单。authorizedAppID 必须来自 pkg/core/application.LoadOwnedApp 的 app.ID，
// 严禁直接取请求体里的 appID。
type MenuScopedCond struct {
	*gormdao.BaseCond
	AppID  string // 必填：已授权的应用 ID
	MenuID string // 单条定位（更新/删除）
}

func (c *MenuScopedCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	if c.MenuID != "" {
		db.Where(clause.Eq{Column: col("id"), Value: c.MenuID})
	}
}

// UpdateMapScoped 在 app_id 约束下按主键更新（WHERE id=? AND app_id=?）。
// app_id 仍写回时也只允许等于 authorizedAppID（调用方应传入不含 app_id 变更的 map）。
func (d *MenuDao) UpdateMapScoped(ctx context.Context, menuID, authorizedAppID string, updateMap map[string]any) error {
	db := d.DB(ctx).Model(&model.MenuEntity{}).Table(d.TableName)
	if err := db.Where("id = ? AND app_id = ?", menuID, authorizedAppID).Updates(updateMap).Error; err != nil {
		return fmt.Errorf("[MenuDao] UpdateMapScoped fail menuID=%s appID=%s: %w", menuID, authorizedAppID, err)
	}
	return nil
}

// DeleteScoped 在 app_id 约束下软删除单条菜单（WHERE id=? AND app_id=?）。
func (d *MenuDao) DeleteScoped(ctx context.Context, menuID, authorizedAppID, deletedBy string) error {
	db := d.DB(ctx).Model(&model.MenuEntity{}).Table(d.TableName)
	updatedField := map[string]any{
		"deleted_at": time.Now(),
		"deleted_by": deletedBy,
	}
	if err := db.Where("id = ? AND app_id = ?", menuID, authorizedAppID).Updates(updatedField).Error; err != nil {
		return fmt.Errorf("[MenuDao] DeleteScoped fail menuID=%s appID=%s: %w", menuID, authorizedAppID, err)
	}
	return nil
}
