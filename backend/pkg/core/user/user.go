// Package user 承载租户内用户（tenant_user）聚合的跨表写能力：
// person 解析/find-or-create + 用户主体 + 部门归属（primary/leader/secondary）的原子写入。
//
// 供 platformadmin（建租户时创建内置管理员）与 tenantadmin（控制台建成员）共用同一实现，
// 避免"平台侧复制一份建用户逻辑"造成的漂移（见 docs/design/system-design.md §5.8）。
package user

import (
	"context"
	"errors"
	"time"

	"github.com/morehao/ark-iam/pkg/core/person"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// 哨兵错误：由调用方（各 app 的 service）映射为自身领域的错误码，
// 公共层不依赖任何具体错误码，避免 pkg 反向依赖 apps。
var (
	ErrNilTx              = errors.New("core/user: tx is required")
	ErrPersonNotFound     = errors.New("core/user: person not found")
	ErrAlreadyInTenant    = errors.New("core/user: person already in tenant")
	ErrDeptLeaderConflict = errors.New("core/user: department already has another leader")
)

// CreateReq 构造 Create 入参。
//
// PersonID 与 Person 二选一：
//   - PersonID 非空：直接关联该已存在自然人（其密码/登录方式不被改动）；
//   - PersonID 为空：使用 Person 在事务内 find-or-create（命中已有自然人同样不改其密码）。
type CreateReq struct {
	TenantID string
	PersonID string
	Person   *person.FindOrCreateReq

	UserType    model.UserType   // 空 => member
	Source      model.UserSource // 空 => manual
	Name        string
	Description string
	Avatar      string
	Status      model.UserStatus // 空 => active
	OwnerType   model.OwnerType  // 空 => normal
	JoinedAt    *time.Time       // 空 => now
	CreatedBy   string

	PrimaryDepartmentID    string   // 行政主部门（唯一，空 = 无主部门）
	SecondaryDepartmentIDs []string // 参与部门（可多条）
	LeaderDepartmentIDs    []string // 负责部门（可多条，每部门至多一个负责人）
}

// Create 在 tx 事务内创建用户主体并建立部门归属关系。
//
// 必须在调用方的事务 tx 内执行（tx 为空返回 ErrNilTx），保证 person/user/部门关系同事务原子。
// 返回：新建的用户实体、本次是否新建了 person（供调用方决定是否回显临时密码）、错误。
// 数据库/网络等系统错误原样上抛（由调用方包装为功能级错误码并记日志）；
// 业务边界（person 不存在、重复入租户、多主部门、负责人冲突）返回上面的哨兵错误。
func Create(ctx context.Context, tx *gorm.DB, req *CreateReq) (*model.UserEntity, bool, error) {
	if tx == nil {
		return nil, false, ErrNilTx
	}

	// 1. 解析 person：显式 personID 直连（校验存在），否则事务内 find-or-create
	personID := req.PersonID
	personCreated := false
	if personID != "" {
		p, err := dao.NewPersonDao().WithTx(tx).GetByID(ctx, personID)
		if err != nil {
			return nil, false, err
		}
		if p == nil || p.ID == "" {
			return nil, false, ErrPersonNotFound
		}
	} else {
		if req.Person == nil {
			return nil, false, ErrPersonNotFound
		}
		p, created, err := person.FindOrCreate(ctx, tx, req.Person)
		if err != nil {
			return nil, false, err
		}
		personID = p.ID
		personCreated = created
	}

	// 2. 同一自然人在本租户内只能有一条 user
	userDao := dao.NewUserDao().WithTx(tx)
	existing, err := userDao.GetListByCond(ctx, &dao.UserCond{TenantID: req.TenantID, PersonID: personID})
	if err != nil {
		return nil, false, err
	}
	if len(existing) > 0 {
		return nil, false, ErrAlreadyInTenant
	}

	// 3. 用户主体
	userType := req.UserType
	if userType == "" {
		userType = model.UserTypeMember
	}
	source := req.Source
	if source == "" {
		source = model.UserSourceManual
	}
	joinedAt := req.JoinedAt
	if joinedAt == nil {
		now := time.Now()
		joinedAt = &now
	}
	status := req.Status
	if status == "" {
		status = model.UserStatusActive
	}
	ownerType := req.OwnerType
	if ownerType == "" {
		ownerType = model.OwnerTypeNormal
	}
	insertEntity := &model.UserEntity{
		TenantID:    req.TenantID,
		PersonID:    personID,
		UserType:    userType,
		Source:      source,
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Status:      status,
		OwnerType:   ownerType,
		JoinedAt:    joinedAt,
		CreatedBy:   req.CreatedBy,
	}
	if err := userDao.Insert(ctx, insertEntity); err != nil {
		return nil, false, err
	}

	// 4. 部门归属：primary（行政主部门，至多 1 条）/ leader（负责人）/ secondary（参与部门）
	if req.PrimaryDepartmentID != "" {
		if err := insertDeptRelation(ctx, tx, req, insertEntity.ID, req.PrimaryDepartmentID, model.DeptUserRelationPrimary); err != nil {
			return nil, false, err
		}
	}
	for _, deptID := range req.LeaderDepartmentIDs {
		if err := ensureDeptLeaderUnique(ctx, tx, req.TenantID, deptID, insertEntity.ID); err != nil {
			return nil, false, err
		}
		if err := insertDeptRelation(ctx, tx, req, insertEntity.ID, deptID, model.DeptUserRelationLeader); err != nil {
			return nil, false, err
		}
	}
	for _, deptID := range req.SecondaryDepartmentIDs {
		if err := insertDeptRelation(ctx, tx, req, insertEntity.ID, deptID, model.DeptUserRelationSecondary); err != nil {
			return nil, false, err
		}
	}
	return insertEntity, personCreated, nil
}

// insertDeptRelation 在事务内写入一条部门归属关系。
func insertDeptRelation(ctx context.Context, tx *gorm.DB, req *CreateReq, userID, deptID string, relationType model.DeptUserRelationType) error {
	return dao.NewDepartmentUserDao().WithTx(tx).Insert(ctx, &model.DepartmentUserEntity{
		TenantID:     req.TenantID,
		DepartmentID: deptID,
		UserID:       userID,
		RelationType: relationType,
		CreatedBy:    req.CreatedBy,
	})
}

// ensureDeptLeaderUnique 保证一个部门至多一个负责人：
// 该部门已有 leader 关系且属于其他用户时返回 ErrDeptLeaderConflict。须在事务内传入 tx。
func ensureDeptLeaderUnique(ctx context.Context, tx *gorm.DB, tenantID, deptID, exceptUserID string) error {
	leaderList, err := dao.NewDepartmentUserDao().WithTx(tx).GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		DepartmentID: deptID,
		RelationType: model.DeptUserRelationLeader,
	})
	if err != nil {
		return err
	}
	for i := range leaderList {
		if leaderList[i].UserID != exceptUserID {
			return ErrDeptLeaderConflict
		}
	}
	return nil
}
