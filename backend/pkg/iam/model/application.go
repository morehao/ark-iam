package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
)

const TableNameApplication = "application"

const (
	AppTypeFirstParty = "first_party"
	AppTypeThirdParty = "third_party"
)

const (
	AppStatusEnable  = "enable"
	AppStatusDisable = "disable"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

type TenantPolicy struct {
	AllowPersonCreateTenant *bool `json:"allowPersonCreateTenant,omitempty"`
	AllowJoinByInvite       *bool `json:"allowJoinByInvite,omitempty"`
}

type ApplicationEntity struct {
	gormdao.BaseEntity
	Code         string         `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex;comment:应用编码" json:"code"`
	TenantPolicy datatypes.JSON `gorm:"column:tenant_policy;type:json;default:'{}';comment:租户策略" json:"tenantPolicy"`
	Name         string         `gorm:"column:name;type:varchar(128);not null;default:'';comment:应用名称" json:"name"`
	Description  string         `gorm:"column:description;type:text;comment:应用描述" json:"description"`
	LogoURL      string         `gorm:"column:logo_url;type:varchar(2048);not null;default:'';comment:应用logo" json:"logoURL"`
	HomepageURL  string         `gorm:"column:homepage_url;type:varchar(2048);not null;default:'';comment:应用主页" json:"homepageURL"`
	Type         string         `gorm:"column:type;type:varchar(32);not null;default:'first_party';comment:应用类型" json:"type"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`
	Visibility   string         `gorm:"column:visibility;type:varchar(32);not null;default:'public';comment:可见性" json:"visibility"`
	Sort         int            `gorm:"column:sort;type:int;not null;default:0;comment:排序" json:"sort"`
	IsSystem     int8           `gorm:"column:is_system;type:smallint;not null;default:0;comment:是否系统内置" json:"isSystem"`
	CreatedBy    string         `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy    string         `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy    string         `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
}

func (ApplicationEntity) TableName() string { return TableNameApplication }

type ApplicationEntityList []ApplicationEntity

func (l ApplicationEntityList) ToMap() map[string]ApplicationEntity {
	m := make(map[string]ApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
