package model

import (
	"gorm.io/gorm"
)

// AllEntities 返回全部 IAM 数据实体，供 AutoMigrate 启动自动建表使用。
// 列表即数据表的单一事实源：新增表时在此追加实体即可，无需再维护 schema SQL。
func AllEntities() []any {
	return []any{
		&TenantEntity{},
		&SystemEntity{},
		&PersonEntity{},
		&UserEntity{},
		&UserIdentityEntity{},
		&UserDepartmentEntity{},
		&UserRoleEntity{},
		&UserLoginLogEntity{},
		&DepartmentEntity{},
		&OrganizationEntity{},
		&OrganizationRoleEntity{},
		&OrganizationUserEntity{},
		&OrganizationRoleUserEntity{},
		&ApplicationEntity{},
		&ApplicationClientEntity{},
		&ApplicationClientSecretEntity{},
		&TenantApplicationEntity{},
		&RoleEntity{},
		&ResourceEntity{},
		&ScopeEntity{},
		&RoleScopeEntity{},
		&MenuEntity{},
		&RoleMenuEntity{},
		&ConnectorEntity{},
		&DomainEntity{},
		&ApiKeyEntity{},
		&RefreshTokenEntity{},
		&SessionAuditEntity{},
		&AuditLogEntity{},
		&LogEntity{},
	}
}

// AutoMigrateAll 基于 GORM AutoMigrate 幂等创建/增量同步全部 IAM 数据表。
// AutoMigrate 只会新增缺失的表/列/索引，不会删除或破坏既有数据，
// 因此可安全地在服务启动时调用（多实例并发调用同样安全）。
func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(AllEntities()...)
}
