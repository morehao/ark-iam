package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameAuditLog = "audit_log"

// AuditAction 审计动作标识（字典值，落 audit_log.action）。
type AuditAction string

// 审计动作取值（禁止硬编码）。
const (
	AuditActionLogin                         AuditAction = "login"                            // 登录
	AuditActionLogout                        AuditAction = "logout"                           // 登出
	AuditActionTenantSwitch                  AuditAction = "tenant.switch"                    // 切换租户
	AuditActionTenantCreate                  AuditAction = "tenant.create"                    // 创建租户
	AuditActionTenantAdminPasswordReset      AuditAction = "tenant.admin_password_reset"      // 重置租户管理员密码
	AuditActionApplicationCreate             AuditAction = "application.create"               // 创建应用
	AuditActionApplicationClientCreate       AuditAction = "application_client.create"        // 创建 OAuth 客户端
	AuditActionApplicationClientCreateSecret AuditAction = "application_client.create_secret" // 创建 OAuth 客户端密钥
	AuditActionApiKeyCreate                  AuditAction = "api_key.create"                   // 创建 API Key
	AuditActionApiKeyRevoke                  AuditAction = "api_key.revoke"                   // 吊销 API Key
)

// AuditResult 审计结果。
type AuditResult string

// 审计结果取值（禁止硬编码）。
const (
	AuditResultSuccess AuditResult = "success" // 成功
	AuditResultFailure AuditResult = "failure" // 失败
)

// AuditTargetType 审计目标类型。
type AuditTargetType string

// 审计目标类型取值（禁止硬编码）。
const (
	AuditTargetTypePerson            AuditTargetType = "person"             // 自然人
	AuditTargetTypeUser              AuditTargetType = "user"               // 租户成员
	AuditTargetTypeTenant            AuditTargetType = "tenant"             // 租户
	AuditTargetTypeApplication       AuditTargetType = "application"        // 应用
	AuditTargetTypeApplicationClient AuditTargetType = "application_client" // 应用客户端
	AuditTargetTypeAPIKey            AuditTargetType = "api_key"            // API Key
)

type AuditLogEntity struct {
	gormdao.BaseEntity
	ActorPersonID string          `gorm:"column:actor_person_id;type:varchar(36);not null;default:'';comment:操作人person id"`
	ActorUserID   string          `gorm:"column:actor_user_id;type:varchar(36);not null;default:'';comment:操作人user id"`
	TenantID      string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	ClientID      string          `gorm:"column:client_id;type:varchar(64);not null;default:'';comment:客户端id"`
	Action        AuditAction     `gorm:"column:action;type:varchar(64);not null;default:'';comment:动作标识"`
	TargetType    AuditTargetType `gorm:"column:target_type;type:varchar(64);not null;default:'';comment:目标类型"`
	TargetID      string          `gorm:"column:target_id;type:varchar(36);not null;default:'';comment:目标id"`
	Result        AuditResult     `gorm:"column:result;type:varchar(16);not null;default:'';comment:结果 success/failure"`
	IP            string          `gorm:"column:ip;type:varchar(64);not null;default:'';comment:IP"`
	UserAgent     string          `gorm:"column:user_agent;type:varchar(512);not null;default:'';comment:UA"`
	Detail        string          `gorm:"column:detail;type:text;comment:详情"`
	CreatedBy     string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy     string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy     string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (AuditLogEntity) TableName() string { return TableNameAuditLog }

type AuditLogEntityList []AuditLogEntity

func (l AuditLogEntityList) ToMap() map[string]AuditLogEntity {
	m := make(map[string]AuditLogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
