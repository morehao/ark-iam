package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
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
	SubtreeUsers(ctx *gin.Context, req *dtotenant.OrganizationSubtreeUsersReq) (*dtotenant.OrganizationSubtreeUsersResp, error)
	GetUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationListReq) (*dtotenant.UserOrganizationListResp, error)
	UpdateUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationsUpdateReq) error
}

type organizationUserSvc struct {
}

var _ OrganizationUserSvc = (*organizationUserSvc)(nil)

func NewOrganizationUserSvc() OrganizationUserSvc {
	return &organizationUserSvc{}
}

func (svc *organizationUserSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationUserCreateReq) (*dtotenant.OrganizationUserCreateResp, error) {
	tenantID := gctx.GetTenantID(ctx)
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
	if err != nil || !organizationVisibleToTenant(orgEntity, tenantID) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}

	insertEntity := &model.OrganizationUserEntity{
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		RelationType:   relationType,
		IsPrimary:      req.IsPrimary,
		CreatedBy:      gctx.GetUserID(ctx),
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
	tenantID := gctx.GetTenantID(ctx)
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
	if err != nil || len(relationList) == 0 {
		return code.GetError(code.OrganizationUserNotExistError)
	}

	userID := gctx.GetUserID(ctx)
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
	tenantID := gctx.GetTenantID(ctx)
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

	userID := gctx.GetUserID(ctx)
	for _, r := range relationList {
		if err := dao.NewOrganizationUserDao().Delete(ctx, r.ID, userID); err != nil {
			glog.Errorf(ctx, "[svcorganizationuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.OrganizationUserDeleteError)
		}
	}
	return nil
}

func (svc *organizationUserSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationUserPageListReq) (*dtotenant.OrganizationUserPageListResp, error) {
	cond := &dao.OrganizationUserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:       gctx.GetTenantID(ctx),
		OrganizationID: req.OrganizationID,
		RelationType:   req.RelationType,
		IsPrimary:      req.IsPrimary,
	}
	relationList, total, err := dao.NewOrganizationUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}

	list := make([]dtotenant.OrganizationUserPageListItem, 0, len(relationList))
	for _, v := range relationList {
		item := dtotenant.OrganizationUserPageListItem{
			OrganizationID: v.OrganizationID,
			UserID:         v.UserID,
			UserName:       svc.userName(ctx, v.UserID),
			RelationType:   v.RelationType,
			IsPrimary:      v.IsPrimary,
		}
		if req.UserName != "" && item.UserName != req.UserName {
			continue
		}
		list = append(list, item)
	}
	return &dtotenant.OrganizationUserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// SubtreeUsers 子树成员聚合：org_path 前缀查子树节点 → 取 member 关系 → 去重用户。
func (svc *organizationUserSvc) SubtreeUsers(ctx *gin.Context, req *dtotenant.OrganizationSubtreeUsersReq) (*dtotenant.OrganizationSubtreeUsersResp, error) {
	tenantID := gctx.GetTenantID(ctx)
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil || !organizationVisibleToTenant(orgEntity, tenantID) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	subList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{
		TenantID: tenantID,
		OrgPath:  orgEntity.OrgPath,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.SubtreeUsers] query subtree fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}
	orgIDs := make([]string, 0, len(subList))
	for _, v := range subList {
		orgIDs = append(orgIDs, v.ID)
	}

	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		RelationType: string(model.OrgUserRelationMember),
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.SubtreeUsers] query relations fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}

	seen := make(map[string]bool)
	list := make([]dtotenant.OrganizationSubtreeUser, 0)
	for _, r := range relationList {
		if containsString(orgIDs, r.OrganizationID) && !seen[r.UserID] {
			seen[r.UserID] = true
			list = append(list, dtotenant.OrganizationSubtreeUser{
				UserID:   r.UserID,
				UserName: svc.userName(ctx, r.UserID),
			})
		}
	}
	return &dtotenant.OrganizationSubtreeUsersResp{List: list}, nil
}

// GetUserOrganizations 用户组织归属（含各节点名称）。
func (svc *organizationUserSvc) GetUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationListReq) (*dtotenant.UserOrganizationListResp, error) {
	tenantID := gctx.GetTenantID(ctx)
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID: tenantID,
		UserID:   req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationuser.GetUserOrganizations] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
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
	return &dtotenant.UserOrganizationListResp{List: list}, nil
}

// UpdateUserOrganizations 全量替换用户归属（member 关系集合，首个为主归属）。
func (svc *organizationUserSvc) UpdateUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationsUpdateReq) error {
	tenantID := gctx.GetTenantID(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
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

	userID := gctx.GetUserID(ctx)
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

// userName 查询用户姓名（关系分页/聚合展示用）。
func (svc *organizationUserSvc) userName(ctx *gin.Context, userID string) string {
	u, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil || u == nil {
		return ""
	}
	return u.Name
}
