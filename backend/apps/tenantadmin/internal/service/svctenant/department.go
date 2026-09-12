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

func departmentVisibleToTenant(entity *model.DepartmentEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

type DepartmentSvc interface {
	Create(ctx *gin.Context, req *dtotenant.DepartmentCreateReq) (*dtotenant.DepartmentCreateResp, error)
	Tree(ctx *gin.Context, req *dtotenant.DepartmentTreeReq) (*dtotenant.DepartmentTreeResp, error)
	Children(ctx *gin.Context, req *dtotenant.DepartmentChildrenReq) (*dtotenant.DepartmentChildrenResp, error)
	Update(ctx *gin.Context, req *dtotenant.DepartmentUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtotenant.DepartmentStatusReq) error
	Delete(ctx *gin.Context, req *dtotenant.DepartmentDeleteReq) error
}

type departmentSvc struct {
}

var _ DepartmentSvc = (*departmentSvc)(nil)

func NewDepartmentSvc() DepartmentSvc {
	return &departmentSvc{}
}

// Create 创建部门节点：根节点 dept_path="/"+id、dept_depth=1；
// 子节点继承父节点路径并做深度上限校验。
func (svc *departmentSvc) Create(ctx *gin.Context, req *dtotenant.DepartmentCreateReq) (*dtotenant.DepartmentCreateResp, error) {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	insertEntity := &model.DepartmentEntity{
		TenantID:  tenantID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		Code:      req.Code,
		Sort:      req.Sort,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserIDString(ctx),
	}
	if insertEntity.Status == "" {
		insertEntity.Status = string(model.DeptNodeStatusActive)
	}

	if req.ParentID != "" {
		parent, err := dao.NewDepartmentDao().GetByID(ctx, req.ParentID)
		if err != nil {
			glog.Errorf(ctx, "[svcdepartment.Create] dao GetByID parent fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.DepartmentCreateError)
		}
		if !departmentVisibleToTenant(parent, tenantID) {
			return nil, code.GetError(code.DepartmentNotExistError)
		}
		if parent.DeptDepth+1 > model.MaxDeptDepth {
			return nil, code.GetError(code.DepartmentCreateError)
		}
		insertEntity.DeptPath = parent.DeptPath
		insertEntity.DeptDepth = parent.DeptDepth + 1
	} else {
		insertEntity.DeptDepth = 1
	}

	// 租户内根节点唯一（每个租户有且仅有一个根，创建租户时自动生成）
	if req.ParentID == "" {
		roots, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{TenantID: tenantID, ParentID: ""})
		if err != nil {
			glog.Errorf(ctx, "[svcdepartment.Create] query root fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.DepartmentCreateError)
		}
		if len(roots) > 0 {
			return nil, code.GetError(code.DepartmentCreateError)
		}
	}

	if err := dao.NewDepartmentDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcdepartment.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentCreateError)
	}
	// 物化路径含自身 ID（ID 由 BeforeCreate 生成，需插入后补写）
	if err := dao.NewDepartmentDao().UpdateMap(ctx, insertEntity.ID, map[string]any{
		"dept_path":  insertEntity.DeptPath + "/" + insertEntity.ID,
		"dept_depth": insertEntity.DeptDepth,
	}); err != nil {
		glog.Errorf(ctx, "[svcdepartment.Create] dao UpdateMap path fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentCreateError)
	}
	return &dtotenant.DepartmentCreateResp{
		DepartmentID: insertEntity.ID,
	}, nil
}

// Tree 部门树：查询租户全量节点（按 sort 升序），应用层组装树。
func (svc *departmentSvc) Tree(ctx *gin.Context, req *dtotenant.DepartmentTreeReq) (*dtotenant.DepartmentTreeResp, error) {
	cond := &dao.DepartmentCond{
		TenantID: gincontext.GetTenantIDString(ctx),
		Name:     req.Name,
		Status:   req.Status,
	}
	deptEntityList, err := dao.NewDepartmentDao().GetListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Tree] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	itemMap := make(map[string]*dtotenant.DepartmentTreeItem, len(deptEntityList))
	for i := range deptEntityList {
		v := &deptEntityList[i]
		itemMap[v.ID] = &dtotenant.DepartmentTreeItem{
			DepartmentID: v.ID,
			ParentID:     v.ParentID,
			DeptPath:     v.DeptPath,
			DeptDepth:    v.DeptDepth,
			CreatedAt:    v.CreatedAt.Unix(),
			DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
				Name:   v.Name,
				Code:   v.Code,
				Sort:   v.Sort,
				Status: v.Status,
			},
			Children: []dtotenant.DepartmentTreeItem{},
		}
	}

	var roots []dtotenant.DepartmentTreeItem
	for _, item := range itemMap {
		if item.ParentID != "" {
			if parent, ok := itemMap[item.ParentID]; ok {
				parent.Children = append(parent.Children, *item)
				continue
			}
		}
		roots = append(roots, *item)
	}
	return &dtotenant.DepartmentTreeResp{List: roots}, nil
}

// Children 某部门直属子部门分页：校验父节点归属租户后按 parentID 查直属子级，
// 并附带每项是否有下级（供前端扁平列表展示层级展开标识）。
func (svc *departmentSvc) Children(ctx *gin.Context, req *dtotenant.DepartmentChildrenReq) (*dtotenant.DepartmentChildrenResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	parent, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Children] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}
	if !departmentVisibleToTenant(parent, tenantID) {
		return nil, code.GetError(code.DepartmentNotExistError)
	}

	cond := &dao.DepartmentCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		ParentID: req.DepartmentID,
		Name:     req.Name,
		Status:   req.Status,
	}
	childList, total, err := dao.NewDepartmentDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Children] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	// 判断每个直属子级是否还有下级（避免 N+1：一次查租户全量 parent_id 集合）
	hasChildSet, err := svc.childDeptIDSet(ctx, tenantID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Children] query child set fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	list := make([]dtotenant.DepartmentChildItem, 0, len(childList))
	for i := range childList {
		v := &childList[i]
		list = append(list, dtotenant.DepartmentChildItem{
			DepartmentID: v.ID,
			ParentID:     v.ParentID,
			DeptDepth:    v.DeptDepth,
			CreatedAt:    v.CreatedAt.Unix(),
			UpdatedAt:    v.UpdatedAt.Unix(),
			HasChildren:  hasChildSet[v.ID],
			DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
				Name:   v.Name,
				Code:   v.Code,
				Sort:   v.Sort,
				Status: v.Status,
			},
		})
	}
	return &dtotenant.DepartmentChildrenResp{List: list, Total: total}, nil
}

// childDeptIDSet 返回租户内"作为父节点存在"的部门 ID 集合（用于 hasChildren 判定）。
func (svc *departmentSvc) childDeptIDSet(ctx *gin.Context, tenantID string) (map[string]bool, error) {
	// 需要查询哪些部门是其他部门的父节点：按 dept_depth > 1 提取 parent_id 集合不可行（分页），
	// 改为查询租户下所有部门，收集其 parent_id。
	list, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{TenantID: tenantID})
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

// Update 全量更新（含移动：改 parentID 时做环路/深度校验并级联更新子树 dept_path/dept_depth）。
func (svc *departmentSvc) Update(ctx *gin.Context, req *dtotenant.DepartmentUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	deptEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	if !departmentVisibleToTenant(deptEntity, tenantID) {
		return code.GetError(code.DepartmentNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	if req.ParentID != deptEntity.ParentID {
		if err := svc.moveNode(ctx, tenantID, deptEntity, req.ParentID, userID); err != nil {
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
	if err := dao.NewDepartmentDao().UpdateMap(ctx, req.DepartmentID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcdepartment.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	return nil
}

// UpdateStatus 局部更新状态（PATCH）。
func (svc *departmentSvc) UpdateStatus(ctx *gin.Context, req *dtotenant.DepartmentStatusReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentUpdateError); err != nil {
		return err
	}
	deptEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.UpdateStatus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	if !departmentVisibleToTenant(deptEntity, gincontext.GetTenantIDString(ctx)) {
		return code.GetError(code.DepartmentNotExistError)
	}
	if req.Status != string(model.DeptNodeStatusActive) && req.Status != string(model.DeptNodeStatusInactive) {
		return code.GetError(code.DepartmentUpdateError)
	}
	if err := dao.NewDepartmentDao().UpdateMap(ctx, req.DepartmentID, map[string]any{
		"status":     req.Status,
		"updated_by": gincontext.GetUserIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcdepartment.UpdateStatus] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	return nil
}

// moveNode 移动节点：环路检测（O(1) 前缀判断）+ 深度校验 + 事务级联更新子树 dept_path/dept_depth。
func (svc *departmentSvc) moveNode(ctx *gin.Context, tenantID string, node *model.DepartmentEntity, newParentID, userID string) error {
	var newPath string
	newDepth := 0
	if newParentID != "" {
		newParent, err := dao.NewDepartmentDao().GetByID(ctx, newParentID)
		if err != nil {
			glog.Errorf(ctx, "[svcdepartment.moveNode] dao GetByID newParent fail, err:%v, nodeID:%s newParentID:%s", err, node.ID, newParentID)
			return code.GetError(code.DepartmentUpdateError)
		}
		if !departmentVisibleToTenant(newParent, tenantID) {
			return code.GetError(code.DepartmentNotExistError)
		}
		// 环路：新父是 node 自身或其子孙（dept_path 前缀判断）
		if newParent.ID == node.ID || strings.HasPrefix(newParent.DeptPath, node.DeptPath) {
			return code.GetError(code.DepartmentUpdateError)
		}
		if newParent.DeptDepth+1 > model.MaxDeptDepth {
			return code.GetError(code.DepartmentUpdateError)
		}
		newPath = newParent.DeptPath + "/" + node.ID
		newDepth = newParent.DeptDepth + 1
	} else {
		newPath = "/" + node.ID
		newDepth = 1
	}

	oldPath, oldDepth := node.DeptPath, node.DeptDepth
	depthDelta := newDepth - oldDepth

	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 节点自身：更新 parent_id/dept_path/dept_depth
		if err := tx.Model(&model.DepartmentEntity{}).
			Where("tenant_id = ? AND id = ?", tenantID, node.ID).
			Updates(map[string]any{
				"parent_id":  newParentID,
				"dept_path":  newPath,
				"dept_depth": newDepth,
				"updated_by": userID,
			}).Error; err != nil {
			return err
		}
		// 子树（不含自身）：dept_path 前缀替换 + dept_depth 平移
		return tx.Model(&model.DepartmentEntity{}).
			Where("tenant_id = ? AND dept_path LIKE ?", tenantID, oldPath+"/%").
			Updates(map[string]any{
				"dept_path":  gorm.Expr("replace(dept_path, ?, ?)", oldPath, newPath),
				"dept_depth": gorm.Expr("dept_depth + ?", depthDelta),
			}).Error
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcdepartment.moveNode] transaction fail, err:%v", txErr)
		return code.GetError(code.DepartmentUpdateError)
	}
	return nil
}

// Delete 删除节点：默认拒绝（有子节点或成员时），?cascade=1 级联软删子树并解绑成员。
func (svc *departmentSvc) Delete(ctx *gin.Context, req *dtotenant.DepartmentDeleteReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.DepartmentDeleteError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	deptEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	if !departmentVisibleToTenant(deptEntity, tenantID) {
		return code.GetError(code.DepartmentNotExistError)
	}

	// 子树节点（含自身）
	subList, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{
		TenantID: tenantID,
		DeptPath: deptEntity.DeptPath,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] query subtree fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	if len(subList) > 1 && !req.Cascade {
		return code.GetError(code.DepartmentDeleteError)
	}
	subDeptIDs := make([]string, 0, len(subList))
	for _, v := range subList {
		subDeptIDs = append(subDeptIDs, v.ID)
	}

	// 成员检查（子树内任一节点有成员则需 cascade）
	memberList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] query members fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	hasMember := false
	for _, m := range memberList {
		if containsString(subDeptIDs, m.DepartmentID) {
			hasMember = true
			break
		}
	}
	if hasMember && !req.Cascade {
		return code.GetError(code.DepartmentDeleteError)
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, v := range subList {
			if err := dao.NewDepartmentDao().Delete(ctx, v.ID, userID); err != nil {
				return err
			}
		}
		for _, m := range memberList {
			if containsString(subDeptIDs, m.DepartmentID) {
				if err := dao.NewDepartmentUserDao().Delete(ctx, m.ID, userID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] transaction fail, err:%v", txErr)
		return code.GetError(code.DepartmentDeleteError)
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
