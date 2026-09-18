package model

import (
	"gorm.io/gorm"
)

// AllEntities 返回全部 IAM 数据实体，供 AutoMigrate 启动自动建表使用。
// 列表即数据表的单一事实源：新增表时在此追加实体即可，无需再维护 schema SQL。
func AllEntities() []any {
	return []any{
		&TenantEntity{},
		&PersonEntity{},
		&UserEntity{},
		&UserIdentityEntity{},
		&UserRoleEntity{},
		&UserLoginLogEntity{},
		&DepartmentEntity{},
		&DepartmentUserEntity{},
		&ApplicationEntity{},
		&ApplicationClientEntity{},
		&ApplicationClientSecretEntity{},
		&TenantApplicationEntity{},
		&RoleEntity{},
		&MenuEntity{},
		&RoleMenuEntity{},
		&ConnectorEntity{},
		&DomainEntity{},
		&ApiKeyEntity{},
		&RefreshTokenEntity{},
		&SessionAuditEntity{},
		&AuditLogEntity{},
		&InviteEntity{},
	}
}

// AutoMigrateAll 基于 GORM AutoMigrate 幂等创建/增量同步全部 IAM 数据表。
// AutoMigrate 只会新增缺失的表/列/索引，不会删除或破坏既有数据，
// 因此可安全地在服务启动时调用（多实例并发调用同样安全）。
func AutoMigrateAll(db *gorm.DB) error {
	if err := db.AutoMigrate(AllEntities()...); err != nil {
		return err
	}
	return EnsurePartialUniqueIndexes(db)
}

// partialUniqueIndexes 是软删除表的部分唯一索引（WHERE deleted_at IS NULL）：
// 普通唯一索引会让"删除后重建同名记录"撞键，部分索引则只约束未删除行。
// GORM AutoMigrate 不支持部分索引，这里在迁移后显式执行（IF NOT EXISTS 幂等）。
// PostgreSQL 与 SQLite（3.8+）均支持该语法。
var partialUniqueIndexes = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_person_username ON person (username) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_person_primary_email ON person (primary_email) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_person_primary_phone ON person (primary_phone) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_identity_issuer_subject ON user_identity (issuer, external_subject) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_invite_code ON tenant_invite (code) WHERE deleted_at IS NULL`,
	// 部门关系唯一性：同一 (租户, 部门, 用户, 关系类型) 只允许一行未删除记录。
	// 关系类型互斥（primary 每用户至多 1 行 / leader 每部门至多 1 人）除服务层校验外，此处提供 DB 级兜底。
	`CREATE UNIQUE INDEX IF NOT EXISTS uk_dept_user ON department_user (tenant_id, department_id, user_id, relation_type) WHERE deleted_at IS NULL`,
}

// EnsurePartialUniqueIndexes 创建/校验软删除表的部分唯一索引（幂等）。
func EnsurePartialUniqueIndexes(db *gorm.DB) error {
	for _, sql := range partialUniqueIndexes {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}
