package svctenant

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type OrganizationUserSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationUserCreateReq) (*dtotenant.OrganizationUserCreateResp, error)
	Update(ctx *gin.Context, req *dtotenant.OrganizationUserUpdateReq) error
	Delete(ctx *gin.Context, req *dtotenant.OrganizationUserDeleteReq) error
	PageList(ctx *gin.Context, req *dtotenant.OrganizationUserPageListReq) (*dtotenant.OrganizationUserPageListResp, error)
}

type organizationUserSvc struct {
}

var _ OrganizationUserSvc = (*organizationUserSvc)(nil)

func NewOrganizationUserSvc() OrganizationUserSvc {
	return &organizationUserSvc{}
}

func (svc *organizationUserSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationUserCreateReq) (*dtotenant.OrganizationUserCreateResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	relationType := req.RelationType
	if relationType == "" {
		relationType = model.OrgUserRelationPrimary
	}
	if !isValidRelationType(relationType) {
		return nil, code.GetError(code.OrganizationUserCreateError)
	}

	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] dao GetByID org fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserCreateError)
	}
	if !organizationVisibleToTenant(orgEntity, tenantID) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] dao GetByID user fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserCreateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}
	// 负责人必须是真实用户：服务账号(机器主体)不能担任部门负责人。
	if relationType == model.OrgUserRelationLeader && userEntity.IsMachine() {
		return nil, code.GetError(code.UserMemberOperationOnlyError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		orgUserDao := dao.NewOrganizationUserDao().WithTx(tx)
		if relationType == model.OrgUserRelationPrimary {
			// primary（行政主部门）每用户至多 1 行：已存在则覆盖其组织归属，避免重复行。
			oldList, err := orgUserDao.GetListByCond(ctx, &dao.OrganizationUserCond{
				TenantID:     tenantID,
				UserID:       req.UserID,
				RelationType: model.OrgUserRelationPrimary,
			})
			if err != nil {
				return err
			}
			if len(oldList) > 0 {
				return orgUserDao.UpdateMap(ctx, oldList[0].ID, map[string]any{
					"organization_id": req.OrganizationID,
					"updated_by":      userID,
				})
			}
		}
		if relationType == model.OrgUserRelationLeader {
			// 一个部门至多一个负责人：冲突拒绝。
			if err := ensureOrgLeaderUnique(ctx, tx, tenantID, req.OrganizationID, req.UserID); err != nil {
				return err
			}
			// 同一用户同一部门 leader 唯一：已存在则幂等跳过。
			exist, err := orgUserDao.GetListByCond(ctx, &dao.OrganizationUserCond{
				TenantID:       tenantID,
				OrganizationID: req.OrganizationID,
				UserID:         req.UserID,
				RelationType:   model.OrgUserRelationLeader,
			})
			if err != nil {
				return err
			}
			if len(exist) > 0 {
				return orgUserDao.UpdateMap(ctx, exist[0].ID, map[string]any{
					"updated_by": userID,
				})
			}
		}
		entity := &model.OrganizationUserEntity{
			TenantID:       tenantID,
			OrganizationID: req.OrganizationID,
			UserID:         req.UserID,
			RelationType:   relationType,
			CreatedBy:      userID,
		}
		return orgUserDao.Insert(ctx, entity)
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserCreateError)
	}
	return &dtotenant.OrganizationUserCreateResp{}, nil
}

func (svc *organizationUserSvc) Update(ctx *gin.Context, req *dtotenant.OrganizationUserUpdateReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	relationType := req.RelationType
	if relationType == "" {
		relationType = model.OrgUserRelationPrimary
	}
	if !isValidRelationType(relationType) {
		return code.GetError(code.OrganizationUserUpdateError)
	}

	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Update] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserUpdateError)
	}
	if len(relationList) == 0 {
		return code.GetError(code.OrganizationUserNotExistError)
	}
	// 负责人必须是真实用户：目标用户为服务账号时禁止收敛为 leader。
	if relationType == model.OrgUserRelationLeader {
		userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
		if err != nil {
			glog.Errorf(ctx, "[svcorganizationuser.Update] dao GetByID user fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.OrganizationUserUpdateError)
		}
		if userEntity != nil && userEntity.IsMachine() {
			return code.GetError(code.UserMemberOperationOnlyError)
		}
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if relationType == model.OrgUserRelationLeader {
			// 一个部门至多一个负责人：目标关系为 leader 且该部门已有其他 leader 时拒绝。
			if err := ensureOrgLeaderUnique(ctx, tx, tenantID, req.OrganizationID, req.UserID); err != nil {
				return err
			}
		}
		// 目标：把该用户在此组织的全部关系收敛为单一关系类型 relationType。
		// 已为目标类型的行跳过，其余行改挂目标类型；若目标类型原本已存在，
		// 其余行直接删除，避免同 org+user+relationType 唯一键冲突。
		hasTarget := false
		for _, r := range relationList {
			if r.RelationType == relationType {
				hasTarget = true
				break
			}
		}
		for _, r := range relationList {
			if r.RelationType == relationType {
				if err := dao.NewOrganizationUserDao().UpdateMap(ctx, r.ID, map[string]any{
					"updated_by": userID,
				}); err != nil {
					return err
				}
				continue
			}
			if hasTarget {
				// 目标类型已存在：删除本行即可收敛
				if err := dao.NewOrganizationUserDao().Delete(ctx, r.ID, userID); err != nil {
					return err
				}
				continue
			}
			// 目标类型不存在：改挂第一行为目标类型
			if err := dao.NewOrganizationUserDao().UpdateMap(ctx, r.ID, map[string]any{
				"relation_type": relationType,
				"updated_by":    userID,
			}); err != nil {
				return err
			}
			hasTarget = true
			if relationType == model.OrgUserRelationPrimary {
				if err := ensureSinglePrimary(ctx, tx, tenantID, req.UserID, r.ID, userID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Update] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserUpdateError)
	}
	return nil
}

// isValidRelationType 校验关系类型是否为合法枚举。
func isValidRelationType(relationType model.OrgUserRelationType) bool {
	switch relationType {
	case model.OrgUserRelationPrimary, model.OrgUserRelationSecondary, model.OrgUserRelationLeader:
		return true
	}
	return false
}

// ensureOrgLeaderUnique 保证一个部门至多一个负责人：查询该部门的 leader 关系，
// 若存在除 exceptUserID 之外的用户则返回负责人冲突错误。须在事务内传入 tx。
func ensureOrgLeaderUnique(ctx *gin.Context, tx *gorm.DB, tenantID, orgID, exceptUserID string) error {
	leaderList, err := dao.NewOrganizationUserDao().WithTx(tx).GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:       tenantID,
		OrganizationID: orgID,
		RelationType:   model.OrgUserRelationLeader,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.ensureOrgLeaderUnique] dao GetListByCond fail, err:%v, orgID:%s", err, orgID)
		return err
	}
	for _, r := range leaderList {
		if r.UserID != exceptUserID {
			return code.GetError(code.OrganizationUserLeaderConflictError)
		}
	}
	return nil
}

// replaceOrgRelationList 全量替换用户某种组织关系集合：删除旧关系并插入 orgIDs 对应新关系。
// 供 user.Create/Update 的组织维度局部/整体更新复用（须在事务内传入 tx）。
func replaceOrgRelationList(ctx *gin.Context, tx *gorm.DB, tenantID, userID string, orgIDs []string, relationType model.OrgUserRelationType, operatorID string) error {
	orgUserDao := dao.NewOrganizationUserDao().WithTx(tx)
	oldList, err := orgUserDao.GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		UserID:       userID,
		RelationType: relationType,
	})
	if err != nil {
		return err
	}
	for _, r := range oldList {
		if err := orgUserDao.Delete(ctx, r.ID, operatorID); err != nil {
			return err
		}
	}
	for _, orgID := range orgIDs {
		if relationType == model.OrgUserRelationLeader {
			// 每部门至多一个负责人：冲突拒绝（删除本用户旧 leader 后，其余用户仍占用则拒绝）。
			if err := ensureOrgLeaderUnique(ctx, tx, tenantID, orgID, userID); err != nil {
				return err
			}
		}
		entity := &model.OrganizationUserEntity{
			TenantID:       tenantID,
			OrganizationID: orgID,
			UserID:         userID,
			RelationType:   relationType,
			CreatedBy:      operatorID,
		}
		if err := orgUserDao.Insert(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// ensureSinglePrimary 保证 primary（行政主部门）每用户至多 1 行：除 exceptID 外不保留其他 primary 行。须在事务内传入 tx。
func ensureSinglePrimary(ctx *gin.Context, tx *gorm.DB, tenantID, userID, exceptID, operatorID string) error {
	orgUserDao := dao.NewOrganizationUserDao().WithTx(tx)
	oldList, err := orgUserDao.GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		UserID:       userID,
		RelationType: model.OrgUserRelationPrimary,
	})
	if err != nil {
		return err
	}
	for _, r := range oldList {
		if r.ID != exceptID {
			if err := orgUserDao.Delete(ctx, r.ID, operatorID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *organizationUserSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationUserDeleteReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Delete] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserDeleteError)
	}
	if len(relationList) == 0 {
		return code.GetError(code.OrganizationUserNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	for _, r := range relationList {
		if err := dao.NewOrganizationUserDao().Delete(ctx, r.ID, userID); err != nil {
			glog.Errorf(ctx, "[svcorganizationuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.OrganizationUserDeleteError)
		}
	}
	return nil
}

func (svc *organizationUserSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationUserPageListReq) (*dtotenant.OrganizationUserPageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.OrganizationUserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		RelationType:   req.RelationType,
	}
	relationList, total, err := dao.NewOrganizationUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}

	// 批量加载用户与其自然人基础信息（消除 N+1）
	userIDs := make([]string, 0, len(relationList))
	for _, v := range relationList {
		userIDs = append(userIDs, v.UserID)
	}
	userMap, personMap := (&userSvc{}).loadUserPersonMaps(ctx, userIDs)

	keyword := req.Keyword
	list := make([]dtotenant.OrganizationUserPageListItem, 0, len(relationList))
	for _, v := range relationList {
		u := userMap[v.UserID]
		if u == nil {
			continue
		}
		person := personMap[u.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		item := dtotenant.OrganizationUserPageListItem{
			OrganizationID: v.OrganizationID,
			UserID:         v.UserID,
			UserType:       u.UserType,
			UserName:       u.Name,
			Username:       model.DerefStr(person.Username),
			PrimaryEmail:   model.DerefStr(person.PrimaryEmail),
			PrimaryPhone:   model.DerefStr(person.PrimaryPhone),
			Avatar:         u.Avatar,
			IsSuspended:    u.IsSuspended,
			RelationType:   v.RelationType,
			JoinedAt:       v.CreatedAt.Unix(),
		}
		if keyword != "" && !matchMemberKeyword(item, keyword) {
			continue
		}
		list = append(list, item)
	}
	return &dtotenant.OrganizationUserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// matchMemberKeyword 成员关键词匹配：姓名/用户名/邮箱/手机任一包含即命中。
func matchMemberKeyword(item dtotenant.OrganizationUserPageListItem, keyword string) bool {
	return strings.Contains(item.UserName, keyword) ||
		strings.Contains(item.Username, keyword) ||
		strings.Contains(item.PrimaryEmail, keyword) ||
		strings.Contains(item.PrimaryPhone, keyword)
}

// loadUserOrganizations 查询用户组织归属（含各节点名称），供用户归属接口与用户详情复用。
func loadUserOrganizations(ctx *gin.Context, tenantID, userID string) ([]dtotenant.UserOrganizationItem, error) {
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.loadUserOrganizations] dao GetListByCond fail, err:%v, userID:%s", err, userID)
		return nil, err
	}

	orgIDSet := make(map[string]bool)
	for _, r := range relationList {
		orgIDSet[r.OrganizationID] = true
	}
	orgNameMap := make(map[string]string, len(orgIDSet))
	for orgID := range orgIDSet {
		if o, err := dao.NewOrganizationDao().GetByID(ctx, orgID); err == nil && o != nil {
			orgNameMap[orgID] = o.Name
		}
	}

	list := make([]dtotenant.UserOrganizationItem, 0, len(relationList))
	for _, r := range relationList {
		list = append(list, dtotenant.UserOrganizationItem{
			OrganizationID:   r.OrganizationID,
			OrganizationName: orgNameMap[r.OrganizationID],
			RelationType:     r.RelationType,
		})
	}
	return list, nil
}
