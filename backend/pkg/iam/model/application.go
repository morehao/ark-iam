package model

import "gorm.io/gorm"

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

type ApplicationEntity struct {
	gorm.Model
	Code        string `gorm:"column:code;type:varchar(64);not null;default '';uniqueIndex;comment:应用编码" json:"code"`
	Name        string `gorm:"column:name;type:varchar(128);not null;default '';comment:应用名称" json:"name"`
	Description string `gorm:"column:description;type:text;comment:应用描述" json:"description"`
	LogoURL     string `gorm:"column:logo_url;type:varchar(2048);not null;default '';comment:应用logo" json:"logoURL"`
	HomepageURL string `gorm:"column:homepage_url;type:varchar(2048);not null;default '';comment:应用主页" json:"homepageURL"`
	Type        string `gorm:"column:type;type:varchar(32);not null;default 'first_party';comment:应用类型" json:"type"`
	Status      string `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`
	Visibility  string `gorm:"column:visibility;type:varchar(32);not null;default 'public';comment:可见性" json:"visibility"`
	Sort        int    `gorm:"column:sort;type:int;not null;default 0;comment:排序" json:"sort"`
	CreatedBy   uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
	UpdatedBy   uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
	DeletedBy   uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationEntity) TableName() string { return TableNameApplication }

type ApplicationEntityList []ApplicationEntity

func (l ApplicationEntityList) ToMap() map[uint]ApplicationEntity {
	m := make(map[uint]ApplicationEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
