package svctenant

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type DepartmentUserSvc interface {
	Create(ctx *gin.Context, req *dtotenant.DepartmentUserCreateReq) (*dtotenant.DepartmentUserCreateResp, error)
	Update(ctx *gin.Context, req *dtotenant.DepartmentUserUpdateReq) error
	Delete(ctx *gin.Context, req *dtotenant.DepartmentUserDeleteReq) error
	PageList(ctx *gin.Context, req *dtotenant.DepartmentUserPageListReq) (*dtotenant.DepartmentUserPageListResp, error)
}

type departmentUserSvc struct {
}

var _ DepartmentUserSvc = (*departmentUserSvc)(nil)

func NewDepartmentUserSvc() DepartmentUserSvc {
	return &departmentUserSvc{}
}

func (svc *departmentUserSvc) Create(ctx *gin.Context, req *dtotenant.DepartmentUserCreateReq) (*dtotenant.DepartmentUserCreateResp, error) {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentUserCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	relationType := req.RelationType
	if relationType == "" {
		relationType = model.DeptUserRelationPrimary
	}
	if !isValidRelationType(relationType) {
		return nil, code.GetError(code.DepartmentUserCreateError)
	}

	deptEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Create] dao GetByID dept fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentUserCreateError)
	}
	if !departmentVisibleToTenant(deptEntity, tenantID) {
		return nil, code.GetError(code.DepartmentNotExistError)
	}
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Create] dao GetByID user fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentUserCreateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}
	// 负责人必须是真实用户：服务账号(机器主体)不能担任部门负责人。
	if relationType == model.DeptUserRelationLeader && userEntity.IsMachine() {
		return nil, code.GetError(code.UserMemberOperationOnlyError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		deptUserDao := dao.NewDepartmentUserDao().WithTx(tx)
		if relationType == model.DeptUserRelationPrimary {
			// primary（行政主部门）每用户至多 1 行：已存在则覆盖其部门归属，避免重复行。
			oldList, err := deptUserDao.GetListByCond(ctx, &dao.DepartmentUserCond{
				TenantID:     tenantID,
				UserID:       req.UserID,
				RelationType: model.DeptUserRelationPrimary,
			})
			if err != nil {
				return err
			}
			if len(oldList) > 0 {
				return deptUserDao.UpdateMap(ctx, oldList[0].ID, map[string]any{
					"department_id": req.DepartmentID,
					"updated_by":    userID,
				})
			}
		}
		if relationType == model.DeptUserRelationLeader {
			// 一个部门至多一个负责人：冲突拒绝。
			if err := ensureDeptLeaderUnique(ctx, tx, tenantID, req.DepartmentID, req.UserID); err != nil {
				return err
			}
			// 同一用户同一部门 leader 唯一：已存在则幂等跳过。
			exist, err := deptUserDao.GetListByCond(ctx, &dao.DepartmentUserCond{
				TenantID:     tenantID,
				DepartmentID: req.DepartmentID,
				UserID:       req.UserID,
				RelationType: model.DeptUserRelationLeader,
			})
			if err != nil {
				return err
			}
			if len(exist) > 0 {
				return deptUserDao.UpdateMap(ctx, exist[0].ID, map[string]any{
					"updated_by": userID,
				})
			}
		}
		entity := &model.DepartmentUserEntity{
			TenantID:     tenantID,
			DepartmentID: req.DepartmentID,
			UserID:       req.UserID,
			RelationType: relationType,
			CreatedBy:    userID,
		}
		return deptUserDao.Insert(ctx, entity)
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentUserCreateError)
	}
	return &dtotenant.DepartmentUserCreateResp{}, nil
}

func (svc *departmentUserSvc) Update(ctx *gin.Context, req *dtotenant.DepartmentUserUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentUserUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	relationType := req.RelationType
	if relationType == "" {
		relationType = model.DeptUserRelationPrimary
	}
	if !isValidRelationType(relationType) {
		return code.GetError(code.DepartmentUserUpdateError)
	}

	relationList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		DepartmentID: req.DepartmentID,
		UserID:       req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Update] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUserUpdateError)
	}
	if len(relationList) == 0 {
		return code.GetError(code.DepartmentUserNotExistError)
	}
	// 负责人必须是真实用户：目标用户为服务账号时禁止收敛为 leader。
	if relationType == model.DeptUserRelationLeader {
		userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
		if err != nil {
			glog.Errorf(ctx, "[svcdepartmentuser.Update] dao GetByID user fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.DepartmentUserUpdateError)
		}
		if userEntity != nil && userEntity.IsMachine() {
			return code.GetError(code.UserMemberOperationOnlyError)
		}
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if relationType == model.DeptUserRelationLeader {
			// 一个部门至多一个负责人：目标关系为 leader 且该部门已有其他 leader 时拒绝。
			if err := ensureDeptLeaderUnique(ctx, tx, tenantID, req.DepartmentID, req.UserID); err != nil {
				return err
			}
		}
		// 目标：把该用户在此部门的全部关系收敛为单一关系类型 relationType。
		// 已为目标类型的行跳过，其余行改挂目标类型；若目标类型原本已存在，
		// 其余行直接删除，避免同 dept+user+relationType 唯一键冲突。
		hasTarget := false
		for _, r := range relationList {
			if r.RelationType == relationType {
				hasTarget = true
				break
			}
		}
		for _, r := range relationList {
			if r.RelationType == relationType {
				if err := dao.NewDepartmentUserDao().UpdateMap(ctx, r.ID, map[string]any{
					"updated_by": userID,
				}); err != nil {
					return err
				}
				continue
			}
			if hasTarget {
				// 目标类型已存在：删除本行即可收敛
				if err := dao.NewDepartmentUserDao().Delete(ctx, r.ID, userID); err != nil {
					return err
				}
				continue
			}
			// 目标类型不存在：改挂第一行为目标类型
			if err := dao.NewDepartmentUserDao().UpdateMap(ctx, r.ID, map[string]any{
				"relation_type": relationType,
				"updated_by":    userID,
			}); err != nil {
				return err
			}
			hasTarget = true
			if relationType == model.DeptUserRelationPrimary {
				if err := ensureSinglePrimary(ctx, tx, tenantID, req.UserID, r.ID, userID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Update] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUserUpdateError)
	}
	return nil
}

// isValidRelationType 校验关系类型是否为合法枚举。
func isValidRelationType(relationType model.DeptUserRelationType) bool {
	switch relationType {
	case model.DeptUserRelationPrimary, model.DeptUserRelationSecondary, model.DeptUserRelationLeader:
		return true
	}
	return false
}

// ensureDeptLeaderUnique 保证一个部门至多一个负责人：查询该部门的 leader 关系，
// 若存在除 exceptUserID 之外的用户则返回负责人冲突错误。须在事务内传入 tx。
func ensureDeptLeaderUnique(ctx *gin.Context, tx *gorm.DB, tenantID, deptID, exceptUserID string) error {
	leaderList, err := dao.NewDepartmentUserDao().WithTx(tx).GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		DepartmentID: deptID,
		RelationType: model.DeptUserRelationLeader,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.ensureDeptLeaderUnique] dao GetListByCond fail, err:%v, deptID:%s", err, deptID)
		return err
	}
	for _, r := range leaderList {
		if r.UserID != exceptUserID {
			return code.GetError(code.DepartmentUserLeaderConflictError)
		}
	}
	return nil
}

// replaceDeptRelationList 全量替换用户某种部门关系集合：删除旧关系并插入 deptIDs 对应新关系。
// 供 user.Create/Update 的部门维度局部/整体更新复用（须在事务内传入 tx）。
func replaceDeptRelationList(ctx *gin.Context, tx *gorm.DB, tenantID, userID string, deptIDs []string, relationType model.DeptUserRelationType, operatorID string) error {
	deptUserDao := dao.NewDepartmentUserDao().WithTx(tx)
	oldList, err := deptUserDao.GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		UserID:       userID,
		RelationType: relationType,
	})
	if err != nil {
		return err
	}
	for _, r := range oldList {
		if err := deptUserDao.Delete(ctx, r.ID, operatorID); err != nil {
			return err
		}
	}
	for _, deptID := range deptIDs {
		if relationType == model.DeptUserRelationLeader {
			// 每部门至多一个负责人：冲突拒绝（删除本用户旧 leader 后，其余用户仍占用则拒绝）。
			if err := ensureDeptLeaderUnique(ctx, tx, tenantID, deptID, userID); err != nil {
				return err
			}
		}
		entity := &model.DepartmentUserEntity{
			TenantID:     tenantID,
			DepartmentID: deptID,
			UserID:       userID,
			RelationType: relationType,
			CreatedBy:    operatorID,
		}
		if err := deptUserDao.Insert(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// ensureSinglePrimary 保证 primary（行政主部门）每用户至多 1 行：除 exceptID 外不保留其他 primary 行。须在事务内传入 tx。
func ensureSinglePrimary(ctx *gin.Context, tx *gorm.DB, tenantID, userID, exceptID, operatorID string) error {
	deptUserDao := dao.NewDepartmentUserDao().WithTx(tx)
	oldList, err := deptUserDao.GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		UserID:       userID,
		RelationType: model.DeptUserRelationPrimary,
	})
	if err != nil {
		return err
	}
	for _, r := range oldList {
		if r.ID != exceptID {
			if err := deptUserDao.Delete(ctx, r.ID, operatorID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *departmentUserSvc) Delete(ctx *gin.Context, req *dtotenant.DepartmentUserDeleteReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentUserDeleteError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	relationList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		DepartmentID: req.DepartmentID,
		UserID:       req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUserDeleteError)
	}
	if len(relationList) == 0 {
		return code.GetError(code.DepartmentUserNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	for _, r := range relationList {
		if err := dao.NewDepartmentUserDao().Delete(ctx, r.ID, userID); err != nil {
			glog.Errorf(ctx, "[svcdepartmentuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.DepartmentUserDeleteError)
		}
	}
	return nil
}

func (svc *departmentUserSvc) PageList(ctx *gin.Context, req *dtotenant.DepartmentUserPageListReq) (*dtotenant.DepartmentUserPageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.DepartmentUserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:     tenantID,
		DepartmentID: req.DepartmentID,
		RelationType: req.RelationType,
	}
	relationList, total, err := dao.NewDepartmentUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentUserGetPageListError)
	}

	// 批量加载用户与其自然人基础信息（消除 N+1）
	userIDs := make([]string, 0, len(relationList))
	for _, v := range relationList {
		userIDs = append(userIDs, v.UserID)
	}
	userMap, personMap := (&userSvc{}).loadUserPersonMaps(ctx, userIDs)

	keyword := req.Keyword
	list := make([]dtotenant.DepartmentUserPageListItem, 0, len(relationList))
	for _, v := range relationList {
		u := userMap[v.UserID]
		if u == nil {
			continue
		}
		person := personMap[u.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		item := dtotenant.DepartmentUserPageListItem{
			DepartmentID: v.DepartmentID,
			UserID:       v.UserID,
			UserType:     u.UserType,
			UserName:     u.Name,
			Username:     model.DerefStr(person.Username),
			PrimaryEmail: model.DerefStr(person.PrimaryEmail),
			PrimaryPhone: model.DerefStr(person.PrimaryPhone),
			Avatar:       u.Avatar,
			IsSuspended:  u.IsSuspended,
			RelationType: v.RelationType,
			JoinedAt:     v.CreatedAt.Unix(),
		}
		if keyword != "" && !matchMemberKeyword(item, keyword) {
			continue
		}
		list = append(list, item)
	}
	return &dtotenant.DepartmentUserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// matchMemberKeyword 成员关键词匹配：姓名/用户名/邮箱/手机任一包含即命中。
func matchMemberKeyword(item dtotenant.DepartmentUserPageListItem, keyword string) bool {
	return strings.Contains(item.UserName, keyword) ||
		strings.Contains(item.Username, keyword) ||
		strings.Contains(item.PrimaryEmail, keyword) ||
		strings.Contains(item.PrimaryPhone, keyword)
}

// loadUserDepartments 查询用户部门归属（含各节点名称），供用户归属接口与用户详情复用。
func loadUserDepartments(ctx *gin.Context, tenantID, userID string) ([]dtotenant.UserDepartmentItem, error) {
	relationList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartmentuser.loadUserDepartments] dao GetListByCond fail, err:%v, userID:%s", err, userID)
		return nil, err
	}

	deptIDSet := make(map[string]bool)
	for _, r := range relationList {
		deptIDSet[r.DepartmentID] = true
	}
	deptNameMap := make(map[string]string, len(deptIDSet))
	for deptID := range deptIDSet {
		if o, err := dao.NewDepartmentDao().GetByID(ctx, deptID); err == nil && o != nil {
			deptNameMap[deptID] = o.Name
		}
	}

	list := make([]dtotenant.UserDepartmentItem, 0, len(relationList))
	for _, r := range relationList {
		list = append(list, dtotenant.UserDepartmentItem{
			DepartmentID:   r.DepartmentID,
			DepartmentName: deptNameMap[r.DepartmentID],
			RelationType:   r.RelationType,
		})
	}
	return list, nil
}
