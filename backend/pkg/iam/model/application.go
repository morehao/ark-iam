package model

import (
	"regexp"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameApplication = "application"

// AppCodePattern 应用编码规则：以小写字母开头，仅含小写字母、数字与下划线（如 platform_admin）。
// 种子内置应用（platform_admin / tenant_admin）与控制台新建应用共用同一口径，禁止连字符。
const AppCodePattern = `^[a-z][a-z0-9_]*$`

var appCodeRegexp = regexp.MustCompile(AppCodePattern)

// IsValidAppCode 判断应用编码是否符合 AppCodePattern。
func IsValidAppCode(code string) bool { return appCodeRegexp.MatchString(code) }

// AppSource 应用来源。
//
// builtin 是 first_party 中受保护的子集（内置应用同样由平台自有），枚举取最具体值：
// 平台随产品交付的控制台应用（管理后台、租户自服务）都落 builtin；
// first_party 留给「平台自建但非内置」的应用（当前种子未产生该来源，控制台也不再创建）。
type AppSource string

// 应用来源取值（禁止硬编码）。
const (
	AppSourceBuiltin    AppSource = "builtin"     // 平台内置：种子播种/随产品交付，禁删，菜单归各自专属控制台
	AppSourceFirstParty AppSource = "first_party" // 平台自建但非内置：可删，菜单并入所在控制台
	AppSourceThirdParty AppSource = "third_party" // 第三方接入：外部/租户接入的应用
)

// IsBuiltin 判断是否为平台内置应用（禁删 + 菜单归专属控制台）。
func (s AppSource) IsBuiltin() bool { return s == AppSourceBuiltin }

// IsPlatformOwned 判断是否平台自有（内置或自建），供「自有 vs 外部」口径使用，避免调用方枚举三值。
func (s AppSource) IsPlatformOwned() bool { return s == AppSourceBuiltin || s == AppSourceFirstParty }

const (
	AppStatusEnable  = "enable"
	AppStatusDisable = "disable"
)

type ApplicationEntity struct {
	gormdao.BaseEntity
	Code                    string    `gorm:"column:code;type:varchar(64);not null;default:'';uniqueIndex;comment:应用编码" json:"code"`
	AllowPersonCreateTenant *bool     `gorm:"column:allow_person_create_tenant;type:boolean;not null;default:false;comment:个人是否可自助创建租户" json:"allowPersonCreateTenant"`
	AllowJoinByInvite       *bool     `gorm:"column:allow_join_by_invite;type:boolean;not null;default:false;comment:是否允许通过邀请加入租户" json:"allowJoinByInvite"`
	Name                    string    `gorm:"column:name;type:varchar(128);not null;default:'';comment:应用名称" json:"name"`
	Description             string    `gorm:"column:description;type:text;comment:应用描述" json:"description"`
	LogoURL                 string    `gorm:"column:logo_url;type:varchar(2048);not null;default:'';comment:应用logo" json:"logoURL"`
	HomepageURL             string    `gorm:"column:homepage_url;type:varchar(2048);not null;default:'';comment:应用主页" json:"homepageURL"`
	Source                  AppSource `gorm:"column:source;type:varchar(32);not null;default:'third_party';comment:应用来源(builtin内置/first_party第一方/third_party第三方)" json:"source"`
	Status                  string    `gorm:"column:status;type:varchar(32);not null;default:'enable';comment:状态" json:"status"`
	Sort                    int       `gorm:"column:sort;type:int;not null;default:0;comment:排序" json:"sort"`
	CreatedBy               string    `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id" json:"createdBy"`
	UpdatedBy               string    `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id" json:"updatedBy"`
	DeletedBy               string    `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id" json:"deletedBy"`
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
