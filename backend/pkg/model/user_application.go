package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUserApplication = "user_application"

// UserApplicationStatus 用户应用准入的启停状态（启停语义统一使用 enable/disable）。
// 撤销准入用 disable 表达（保留来源与审计痕迹），软删只用于实体下线。
type UserApplicationStatus string

// 用户应用准入状态取值（禁止硬编码）。
const (
	UserApplicationStatusEnable  UserApplicationStatus = "enable"  // 已准入
	UserApplicationStatusDisable UserApplicationStatus = "disable" // 已撤销
)

// UserApplicationSource 用户应用准入的来源，回答"这条准入是谁写的"，供审计与运维排查。
type UserApplicationSource string

// 用户应用准入来源取值（禁止硬编码）。
const (
	UserApplicationSourceManual    UserApplicationSource = "manual"    // 租户管理员手工分配
	UserApplicationSourceRole      UserApplicationSource = "role"      // 角色授予时自动补齐（授权⇒准入）
	UserApplicationSourceProvision UserApplicationSource = "provision" // 引导/建租户随内置管理员写入
)

// UserApplicationEntity 用户应用准入：租户内「成员 × 应用」的唯一准入事实源。
//
// 语义边界：准入只回答"该成员能否使用某应用"，是登录与令牌签发的必要条件；
// 进入应用后能做什么由角色（user_role/role_menu）决定，两者互不替代。
// 完整判定 = tenant_application.status=enable ∧ application.status=enable ∧ 本表 status=enable。
//
// UserID 指向 tenant_user.id（租户成员主键，真实用户与服务账号同一口径），不是 person.id：
// 协议层只拿得到 person，须先解析出成员行再判定（同 user_role.user_id 口径）。
type UserApplicationEntity struct {
	gormdao.BaseEntity
	TenantID  string                `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id;uniqueIndex:uk_user_application_active,where:deleted_at IS NULL" json:"tenantID"`
	UserID    string                `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:租户成员id(tenant_user.id);uniqueIndex:uk_user_application_active,where:deleted_at IS NULL;index:idx_user_application_app,priority:2" json:"userID"`
	AppID     string                `gorm:"column:app_id;type:varchar(36);not null;default:'';comment:应用id;uniqueIndex:uk_user_application_active,where:deleted_at IS NULL;index:idx_user_application_app,priority:1" json:"appID"`
	Status    UserApplicationStatus `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:准入状态(enable/disable)" json:"status"`
	Source    UserApplicationSource `gorm:"column:source;type:varchar(32);not null;default:'manual';comment:准入来源(manual/role/provision)" json:"source"`
	CreatedBy string                `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy string                `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy string                `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (UserApplicationEntity) TableName() string { return TableNameUserApplication }

type UserApplicationEntityList []UserApplicationEntity

func (l UserApplicationEntityList) ToMap() map[string]UserApplicationEntity {
	m := make(map[string]UserApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
