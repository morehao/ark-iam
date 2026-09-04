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
	"github.com/morehao/golib/dbaccess/gormdao"
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
	Children(ctx *gin.Context, req *dtotenant.OrganizationChildrenReq) (*dtotenant.OrganizationChildrenResp, error)
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
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.OrganizationCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	insertEntity := &model.OrganizationEntity{
		TenantID:  tenantID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Code:      req.Code,
		Sort:      req.Sort,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserIDString(ctx),
	}
	if insertEntity.Status == "" {
		insertEntity.Status = string(model.OrgNodeStatusActive)
	}

	if req.ParentID != "" {
		parent, err := dao.NewOrganizationDao().GetByID(ctx, req.ParentID)
		if err != nil {
			glog.Errorf(ctx, "[svcorganization.Create] dao GetByID parent fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.OrganizationCreateError)
		}
		if !organizationVisibleToTenant(parent, tenantID) {
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
		TenantID: gincontext.GetTenantIDString(ctx),
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
			CreatedAt:      v.CreatedAt.Unix(),
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

// Children 某部门直属子部门分页：校验父节点归属租户后按 parentID 查直属子级，
// 并附带每项是否有下级（供前端扁平列表展示层级展开标识）。
func (svc *organizationSvc) Children(ctx *gin.Context, req *dtotenant.OrganizationChildrenReq) (*dtotenant.OrganizationChildrenResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	parent, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Children] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}
	if !organizationVisibleToTenant(parent, tenantID) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	cond := &dao.OrganizationCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		ParentID: req.OrganizationID,
		Name:     req.Name,
		Status:   req.Status,
	}
	childList, total, err := dao.NewOrganizationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Children] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}

	// 判断每个直属子级是否还有下级（避免 N+1：一次查租户全量 parent_id 集合）
	hasChildSet, err := svc.childOrgIDSet(ctx, tenantID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Children] query child set fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}

	list := make([]dtotenant.OrganizationChildItem, 0, len(childList))
	for i := range childList {
		v := &childList[i]
		list = append(list, dtotenant.OrganizationChildItem{
			OrganizationID: v.ID,
			ParentID:       v.ParentID,
			OrgDepth:       v.OrgDepth,
			CreatedAt:      v.CreatedAt.Unix(),
			HasChildren:    hasChildSet[v.ID],
			OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
				Name:   v.Name,
				Code:   v.Code,
				Sort:   v.Sort,
				Status: v.Status,
			},
		})
	}
	return &dtotenant.OrganizationChildrenResp{List: list, Total: total}, nil
}

// childOrgIDSet 返回租户内"作为父节点存在"的组织 ID 集合（用于 hasChildren 判定）。
func (svc *organizationSvc) childOrgIDSet(ctx *gin.Context, tenantID string) (map[string]bool, error) {
	// 需要查询哪些组织是其他组织的父节点：按 org_depth > 1 提取 parent_id 集合不可行（分页），
	// 改为查询租户下所有组织，收集其 parent_id。
	list, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(list))
	for _, v := range list {
		if v.ParentID != "" {
			set[v.ParentID] = true
		}
	}
	return set, nil
}

// Update 全量更新（含移动：改 parentID 时做环路/深度校验并级联更新子树 org_path/org_depth）。
func (svc *organizationSvc) Update(ctx *gin.Context, req *dtotenant.OrganizationUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.OrganizationUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if !organizationVisibleToTenant(orgEntity, tenantID) {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
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
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.OrganizationUpdateError); err != nil {
		return err
	}
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.UpdateStatus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantIDString(ctx)) {
		return code.GetError(code.OrganizationNotExistError)
	}
	if req.Status != string(model.OrgNodeStatusActive) && req.Status != string(model.OrgNodeStatusInactive) {
		return code.GetError(code.OrganizationUpdateError)
	}
	if err := dao.NewOrganizationDao().UpdateMap(ctx, req.OrganizationID, map[string]any{
		"status":     req.Status,
		"updated_by": gincontext.GetUserIDString(ctx),
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
		if err != nil {
			glog.Errorf(ctx, "[svcorganization.moveNode] dao GetByID newParent fail, err:%v, nodeID:%s newParentID:%s", err, node.ID, newParentID)
			return code.GetError(code.OrganizationUpdateError)
		}
		if !organizationVisibleToTenant(newParent, tenantID) {
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
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.OrganizationDeleteError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
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

	userID := gincontext.GetUserIDString(ctx)
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
