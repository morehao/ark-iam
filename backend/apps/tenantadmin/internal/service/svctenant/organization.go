package svctenant

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

func organizationVisibleToTenant(entity *model.OrganizationEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

type OrganizationSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationCreateReq) (*dtotenant.OrganizationCreateResp, error)
	Tree(ctx *gin.Context, req *dtotenant.OrganizationTreeReq) (*dtotenant.OrganizationTreeResp, error)
	Detail(ctx *gin.Context, req *dtotenant.OrganizationDetailReq) (*dtotenant.OrganizationDetailResp, error)
	Update(ctx *gin.Context, req *dtotenant.OrganizationUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtotenant.OrganizationStatusReq) error
	Delete(ctx *gin.Context, req *dtotenant.OrganizationDeleteReq) error
}

type organizationSvc struct {
}

var _ OrganizationSvc = (*organizationSvc)(nil)

func NewOrganizationSvc() OrganizationSvc {
	return &organizationSvc{}
}

// Create 创建组织节点：根节点 org_path="/"+id、org_depth=1；
// 子节点继承父节点路径并做深度上限校验。
func (svc *organizationSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationCreateReq) (*dtotenant.OrganizationCreateResp, error) {
	tenantID := gincontext.GetTenantID(ctx)
	insertEntity := &model.OrganizationEntity{
		TenantID:  tenantID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Code:      req.Code,
		Sort:      req.Sort,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserID(ctx),
	}
	if insertEntity.Status == "" {
		insertEntity.Status = string(model.OrgNodeStatusActive)
	}

	if req.ParentID != "" {
		parent, err := dao.NewOrganizationDao().GetByID(ctx, req.ParentID)
		if err != nil || !organizationVisibleToTenant(parent, tenantID) {
			return nil, code.GetError(code.OrganizationNotExistError)
		}
		if parent.OrgDepth+1 > model.MaxOrgDepth {
			return nil, code.GetError(code.OrganizationCreateError)
		}
		insertEntity.OrgPath = parent.OrgPath
		insertEntity.OrgDepth = parent.OrgDepth + 1
	} else {
		insertEntity.OrgDepth = 1
	}

	// 租户内根节点唯一（每个租户有且仅有一个根，创建租户时自动生成）
	if req.ParentID == "" {
		roots, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID, ParentID: ""})
		if err != nil {
			glog.Errorf(ctx, "[svcorganization.Create] query root fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.OrganizationCreateError)
		}
		if len(roots) > 0 {
			return nil, code.GetError(code.OrganizationCreateError)
		}
	}

	if err := dao.NewOrganizationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganization.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationCreateError)
	}
	// 物化路径含自身 ID（ID 由 BeforeCreate 生成，需插入后补写）
	if err := dao.NewOrganizationDao().UpdateMap(ctx, insertEntity.ID, map[string]any{
		"org_path":  insertEntity.OrgPath + "/" + insertEntity.ID,
		"org_depth": insertEntity.OrgDepth,
	}); err != nil {
		glog.Errorf(ctx, "[svcorganization.Create] dao UpdateMap path fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationCreateError)
	}
	return &dtotenant.OrganizationCreateResp{
		OrganizationID: insertEntity.ID,
	}, nil
}

// Tree 组织树：查询租户全量节点（按 sort 升序），应用层组装树。
func (svc *organizationSvc) Tree(ctx *gin.Context, req *dtotenant.OrganizationTreeReq) (*dtotenant.OrganizationTreeResp, error) {
	cond := &dao.OrganizationCond{
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Status:   req.Status,
	}
	orgEntityList, err := dao.NewOrganizationDao().GetListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Tree] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}

	itemMap := make(map[string]*dtotenant.OrganizationTreeItem, len(orgEntityList))
	for i := range orgEntityList {
		v := &orgEntityList[i]
		itemMap[v.ID] = &dtotenant.OrganizationTreeItem{
			OrganizationID: v.ID,
			ParentID:       v.ParentID,
			OrgPath:        v.OrgPath,
			OrgDepth:       v.OrgDepth,
			OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
				Name:   v.Name,
				Code:   v.Code,
				Sort:   v.Sort,
				Status: v.Status,
			},
			Children: []dtotenant.OrganizationTreeItem{},
		}
	}

	var roots []dtotenant.OrganizationTreeItem
	for _, item := range itemMap {
		if item.ParentID != "" {
			if parent, ok := itemMap[item.ParentID]; ok {
				parent.Children = append(parent.Children, *item)
				continue
			}
		}
		roots = append(roots, *item)
	}
	return &dtotenant.OrganizationTreeResp{List: roots}, nil
}

// Detail 节点详情（含祖先链面包屑）。
func (svc *organizationSvc) Detail(ctx *gin.Context, req *dtotenant.OrganizationDetailReq) (*dtotenant.OrganizationDetailResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	resp := &dtotenant.OrganizationDetailResp{
		OrganizationID: orgEntity.ID,
		ParentID:       orgEntity.ParentID,
		OrgPath:        orgEntity.OrgPath,
		OrgDepth:       orgEntity.OrgDepth,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
			Name:   orgEntity.Name,
			Code:   orgEntity.Code,
			Sort:   orgEntity.Sort,
			Status: orgEntity.Status,
		},
	}
	resp.Ancestors = svc.loadAncestors(ctx, orgEntity.OrgPath, gincontext.GetTenantID(ctx))
	return resp, nil
}

// loadAncestors 按 org_path 解析祖先链（不含自身），批量查名后自顶向下排序。
func (svc *organizationSvc) loadAncestors(ctx *gin.Context, orgPath, tenantID string) []dtotenant.OrganizationAncestor {
	if orgPath == "" {
		return nil
	}
	parts := strings.Split(strings.Trim(orgPath, "/"), "/")
	var ids []string
	for _, p := range parts {
		if p != "" {
			ids = append(ids, p)
		}
	}
	if len(ids) <= 1 {
		return nil
	}
	ancestorIDs := ids[:len(ids)-1]
	list, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
	if err != nil {
		return nil
	}
	nameMap := make(map[string]string, len(list))
	for _, v := range list {
		nameMap[v.ID] = v.Name
	}
	ancestors := make([]dtotenant.OrganizationAncestor, 0, len(ancestorIDs))
	for _, id := range ancestorIDs {
		ancestors = append(ancestors, dtotenant.OrganizationAncestor{OrganizationID: id, Name: nameMap[id]})
	}
	return ancestors
}

// Update 全量更新（含移动：改 parentID 时做环路/深度校验并级联更新子树 org_path/org_depth）。
func (svc *organizationSvc) Update(ctx *gin.Context, req *dtotenant.OrganizationUpdateReq) error {
	tenantID := gincontext.GetTenantID(ctx)
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if !organizationVisibleToTenant(orgEntity, tenantID) {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if req.ParentID != orgEntity.ParentID {
		if err := svc.moveNode(ctx, tenantID, orgEntity, req.ParentID, userID); err != nil {
			return err
		}
	}
	updateMap := map[string]any{
		"name":       req.Name,
		"code":       req.Code,
		"sort":       req.Sort,
		"status":     req.Status,
		"updated_by": userID,
	}
	if err := dao.NewOrganizationDao().UpdateMap(ctx, req.OrganizationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	return nil
}

// UpdateStatus 局部更新状态（PATCH）。
func (svc *organizationSvc) UpdateStatus(ctx *gin.Context, req *dtotenant.OrganizationStatusReq) error {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.UpdateStatus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OrganizationNotExistError)
	}
	if req.Status != string(model.OrgNodeStatusActive) && req.Status != string(model.OrgNodeStatusInactive) {
		return code.GetError(code.OrganizationUpdateError)
	}
	if err := dao.NewOrganizationDao().UpdateMap(ctx, req.OrganizationID, map[string]any{
		"status":     req.Status,
		"updated_by": gincontext.GetUserID(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcorganization.UpdateStatus] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	return nil
}

// moveNode 移动节点：环路检测（O(1) 前缀判断）+ 深度校验 + 事务级联更新子树 org_path/org_depth。
func (svc *organizationSvc) moveNode(ctx *gin.Context, tenantID string, node *model.OrganizationEntity, newParentID, userID string) error {
	var newPath string
	newDepth := 0
	if newParentID != "" {
		newParent, err := dao.NewOrganizationDao().GetByID(ctx, newParentID)
		if err != nil || !organizationVisibleToTenant(newParent, tenantID) {
			return code.GetError(code.OrganizationNotExistError)
		}
		// 环路：新父是 node 自身或其子孙（org_path 前缀判断）
		if newParent.ID == node.ID || strings.HasPrefix(newParent.OrgPath, node.OrgPath) {
			return code.GetError(code.OrganizationUpdateError)
		}
		if newParent.OrgDepth+1 > model.MaxOrgDepth {
			return code.GetError(code.OrganizationUpdateError)
		}
		newPath = newParent.OrgPath + "/" + node.ID
		newDepth = newParent.OrgDepth + 1
	} else {
		newPath = "/" + node.ID
		newDepth = 1
	}

	oldPath, oldDepth := node.OrgPath, node.OrgDepth
	depthDelta := newDepth - oldDepth

	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 节点自身：更新 parent_id/org_path/org_depth
		if err := tx.Model(&model.OrganizationEntity{}).
			Where("tenant_id = ? AND id = ?", tenantID, node.ID).
			Updates(map[string]any{
				"parent_id":  newParentID,
				"org_path":   newPath,
				"org_depth":  newDepth,
				"updated_by": userID,
			}).Error; err != nil {
			return err
		}
		// 子树（不含自身）：org_path 前缀替换 + org_depth 平移
		return tx.Model(&model.OrganizationEntity{}).
			Where("tenant_id = ? AND org_path LIKE ?", tenantID, oldPath+"/%").
			Updates(map[string]any{
				"org_path":  gorm.Expr("replace(org_path, ?, ?)", oldPath, newPath),
				"org_depth": gorm.Expr("org_depth + ?", depthDelta),
			}).Error
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganization.moveNode] transaction fail, err:%v", txErr)
		return code.GetError(code.OrganizationUpdateError)
	}
	return nil
}

// Delete 删除节点：默认拒绝（有子节点或成员时），?cascade=1 级联软删子树并解绑成员。
func (svc *organizationSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationDeleteReq) error {
	tenantID := gincontext.GetTenantID(ctx)
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	if !organizationVisibleToTenant(orgEntity, tenantID) {
		return code.GetError(code.OrganizationNotExistError)
	}

	// 子树节点（含自身）
	subList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{
		TenantID: tenantID,
		OrgPath:  orgEntity.OrgPath,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] query subtree fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	if len(subList) > 1 && !req.Cascade {
		return code.GetError(code.OrganizationDeleteError)
	}
	subOrgIDs := make([]string, 0, len(subList))
	for _, v := range subList {
		subOrgIDs = append(subOrgIDs, v.ID)
	}

	// 成员检查（子树内任一节点有成员则需 cascade）
	memberList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] query members fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	hasMember := false
	for _, m := range memberList {
		if containsString(subOrgIDs, m.OrganizationID) {
			hasMember = true
			break
		}
	}
	if hasMember && !req.Cascade {
		return code.GetError(code.OrganizationDeleteError)
	}

	userID := gincontext.GetUserID(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, v := range subList {
			if err := dao.NewOrganizationDao().Delete(ctx, v.ID, userID); err != nil {
				return err
			}
		}
		for _, m := range memberList {
			if containsString(subOrgIDs, m.OrganizationID) {
				if err := dao.NewOrganizationUserDao().Delete(ctx, m.ID, userID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] transaction fail, err:%v", txErr)
		return code.GetError(code.OrganizationDeleteError)
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
