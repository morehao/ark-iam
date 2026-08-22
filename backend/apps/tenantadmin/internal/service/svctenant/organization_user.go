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
	UpdateUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationsUpdateReq) error
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
		relationType = string(model.OrgUserRelationMember)
	}
	if relationType != string(model.OrgUserRelationMember) && relationType != string(model.OrgUserRelationLeader) {
		return nil, code.GetError(code.OrganizationUserCreateError)
	}
	if req.IsPrimary && relationType != string(model.OrgUserRelationMember) {
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

	insertEntity := &model.OrganizationUserEntity{
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		RelationType:   relationType,
		IsPrimary:      req.IsPrimary,
		CreatedBy:      gincontext.GetUserIDString(ctx),
	}

	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if insertEntity.IsPrimary {
			// 主归属唯一：先清该用户现有主归属
			if err := clearPrimaryOrg(ctx, tenantID, req.UserID); err != nil {
				return err
			}
		}
		return dao.NewOrganizationUserDao().Insert(ctx, insertEntity)
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
		relationType = string(model.OrgUserRelationMember)
	}
	if relationType != string(model.OrgUserRelationMember) && relationType != string(model.OrgUserRelationLeader) {
		return code.GetError(code.OrganizationUserUpdateError)
	}
	if req.IsPrimary && relationType != string(model.OrgUserRelationMember) {
		return code.GetError(code.OrganizationUserUpdateError)
	}

	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		RelationType:   relationType,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Update] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserUpdateError)
	}
	if len(relationList) == 0 {
		return code.GetError(code.OrganizationUserNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if req.IsPrimary {
			if err := clearPrimaryOrg(ctx, tenantID, req.UserID); err != nil {
				return err
			}
		}
		return dao.NewOrganizationUserDao().UpdateMap(ctx, relationList[0].ID, map[string]any{
			"relation_type": relationType,
			"is_primary":    req.IsPrimary,
			"updated_by":    userID,
		})
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganizationuser.Update] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserUpdateError)
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
		IsPrimary:      req.IsPrimary,
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
			UserName:       u.Name,
			Username:       model.DerefStr(person.Username),
			PrimaryEmail:   model.DerefStr(person.PrimaryEmail),
			PrimaryPhone:   model.DerefStr(person.PrimaryPhone),
			Avatar:         u.Avatar,
			IsSuspended:    u.IsSuspended,
			RelationType:   v.RelationType,
			IsPrimary:      v.IsPrimary,
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
			IsPrimary:        r.IsPrimary,
		})
	}
	return list, nil
}

// UpdateUserOrganizations 全量替换用户归属（member 关系集合，首个为主归属）。
func (svc *organizationUserSvc) UpdateUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationsUpdateReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	// 用户必须从属于至少一个部门（业务约束，防绕过 DTO 校验）
	if len(req.OrganizationIDs) == 0 {
		return code.GetError(code.UserOrganizationRequiredError)
	}
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.UpdateUserOrganizations] dao GetByID user fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserCreateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return code.GetError(code.UserNotExistError)
	}
	orgList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.UpdateUserOrganizations] query org fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserCreateError)
	}
	orgSet := make(map[string]bool, len(orgList))
	for _, o := range orgList {
		orgSet[o.ID] = true
	}
	for _, orgID := range req.OrganizationIDs {
		if !orgSet[orgID] {
			return code.GetError(code.OrganizationNotExistError)
		}
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		oldList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
			TenantID:     tenantID,
			UserID:       req.UserID,
			RelationType: string(model.OrgUserRelationMember),
		})
		if err != nil {
			return err
		}
		for _, r := range oldList {
			if err := dao.NewOrganizationUserDao().Delete(ctx, r.ID, userID); err != nil {
				return err
			}
		}
		for i, orgID := range req.OrganizationIDs {
			entity := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         req.UserID,
				RelationType:   string(model.OrgUserRelationMember),
				IsPrimary:      i == 0,
				CreatedBy:      userID,
			}
			if err := dao.NewOrganizationUserDao().Insert(ctx, entity); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganizationuser.UpdateUserOrganizations] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUserCreateError)
	}
	return nil
}

// clearPrimaryOrg 清除用户现有主归属标记。
func clearPrimaryOrg(ctx *gin.Context, tenantID, userID string) error {
	oldList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:  tenantID,
		UserID:    userID,
		IsPrimary: boolPtr(true),
	})
	if err != nil {
		return err
	}
	for _, r := range oldList {
		if err := dao.NewOrganizationUserDao().UpdateMap(ctx, r.ID, map[string]any{"is_primary": false}); err != nil {
			return err
		}
	}
	return nil
}

func boolPtr(b bool) *bool { return &b }
