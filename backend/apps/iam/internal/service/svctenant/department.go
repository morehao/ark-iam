package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type departmentScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.DepartmentEntity, error)
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.DepartmentEntityList, int64, error)
}

var newDepartmentScopeRepo = func() departmentScopeRepository {
	return dao.NewDepartmentDao()
}

func departmentVisibleToTenant(entity *model.DepartmentEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

type DepartmentSvc interface {
	Create(ctx *gin.Context, req *dtotenant.DepartmentCreateReq) (*dtotenant.DepartmentCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.DepartmentDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.DepartmentUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.DepartmentDetailReq) (*dtotenant.DepartmentDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.DepartmentPageListReq) (*dtotenant.DepartmentPageListResp, error)
	Tree(ctx *gin.Context, req *dtotenant.DepartmentTreeReq) (*dtotenant.DepartmentTreeResp, error)
}

type departmentSvc struct {
}

var _ DepartmentSvc = (*departmentSvc)(nil)

func NewDepartmentSvc() DepartmentSvc {
	return &departmentSvc{}
}

func (svc *departmentSvc) Create(ctx *gin.Context, req *dtotenant.DepartmentCreateReq) (*dtotenant.DepartmentCreateResp, error) {
	insertEntity := &model.DepartmentEntity{
		TenantID:     req.TenantID,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Code:         req.Code,
		Sort:         req.Sort,
		LeaderUserID: req.LeaderUserID,
		CreatedBy:    gincontext.GetUserID(ctx),
	}

	if err := dao.NewDepartmentDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svctenant.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentCreateError)
	}
	return &dtotenant.DepartmentCreateResp{
		DepartmentID: insertEntity.ID,
	}, nil
}

func (svc *departmentSvc) Delete(ctx *gin.Context, req *dtotenant.DepartmentDeleteReq) error {
	departmentEntity, err := newDepartmentScopeRepo().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	if !departmentVisibleToTenant(departmentEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.DepartmentNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewDepartmentDao().Delete(ctx, req.DepartmentID, userID); err != nil {
		glog.Errorf(ctx, "[svctenant.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	return nil
}

func (svc *departmentSvc) Update(ctx *gin.Context, req *dtotenant.DepartmentUpdateReq) error {
	departmentEntity, err := newDepartmentScopeRepo().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	if !departmentVisibleToTenant(departmentEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.DepartmentNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":      req.TenantID,
		"parent_id":      req.ParentID,
		"name":           req.Name,
		"code":           req.Code,
		"sort":           req.Sort,
		"leader_user_id": req.LeaderUserID,
		"updated_by":     userID,
	}
	if err := dao.NewDepartmentDao().UpdateMap(ctx, req.DepartmentID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenant.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	return nil
}

func (svc *departmentSvc) Detail(ctx *gin.Context, req *dtotenant.DepartmentDetailReq) (*dtotenant.DepartmentDetailResp, error) {
	departmentEntity, err := newDepartmentScopeRepo().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetDetailError)
	}
	if !departmentVisibleToTenant(departmentEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.DepartmentNotExistError)
	}

	resp := &dtotenant.DepartmentDetailResp{
		DepartmentID: departmentEntity.ID,
		DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
			TenantID:     departmentEntity.TenantID,
			ParentID:     departmentEntity.ParentID,
			Name:         departmentEntity.Name,
			Code:         departmentEntity.Code,
			Sort:         departmentEntity.Sort,
			LeaderUserID: departmentEntity.LeaderUserID,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: departmentEntity.CreatedAt.Unix(),
			UpdatedAt: departmentEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *departmentSvc) PageList(ctx *gin.Context, req *dtotenant.DepartmentPageListReq) (*dtotenant.DepartmentPageListResp, error) {
	departmentRepo := newDepartmentScopeRepo()
	cond := &dao.DepartmentCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		ParentID: req.ParentID,
		Name:     req.Name,
		Code:     req.Code,
	}
	departmentEntityList, total, err := departmentRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	list := make([]dtotenant.DepartmentPageListItem, 0, len(departmentEntityList))
	for _, v := range departmentEntityList {
		list = append(list, dtotenant.DepartmentPageListItem{
			DepartmentID: v.ID,
			DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
				TenantID:     v.TenantID,
				ParentID:     v.ParentID,
				Name:         v.Name,
				Code:         v.Code,
				Sort:         v.Sort,
				LeaderUserID: v.LeaderUserID,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.DepartmentPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *departmentSvc) Tree(ctx *gin.Context, req *dtotenant.DepartmentTreeReq) (*dtotenant.DepartmentTreeResp, error) {
	departmentRepo := newDepartmentScopeRepo()
	cond := &dao.DepartmentCond{
		TenantID: gincontext.GetTenantID(ctx),
	}
	departmentEntityList, _, err := departmentRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.Tree] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	deptMap := make(map[uint]*model.DepartmentEntity)
	for i := range departmentEntityList {
		deptMap[departmentEntityList[i].ID] = &departmentEntityList[i]
	}

	var buildTree func(parentID uint) []dtotenant.DepartmentTreeItem
	buildTree = func(parentID uint) []dtotenant.DepartmentTreeItem {
		var items []dtotenant.DepartmentTreeItem
		for _, dept := range departmentEntityList {
			if dept.ParentID == parentID {
				item := dtotenant.DepartmentTreeItem{
					DepartmentID: dept.ID,
					DepartmentBaseInfo: objtenant.DepartmentBaseInfo{
						TenantID:     dept.TenantID,
						ParentID:     dept.ParentID,
						Name:         dept.Name,
						Code:         dept.Code,
						Sort:         dept.Sort,
						LeaderUserID: dept.LeaderUserID,
					},
					OperatorBaseInfo: gobject.OperatorBaseInfo{
						UpdatedAt: dept.UpdatedAt.Unix(),
					},
					Children: buildTree(dept.ID),
				}
				items = append(items, item)
			}
		}
		_ = deptMap
		return items
	}

	return &dtotenant.DepartmentTreeResp{
		List: buildTree(0),
	}, nil
}
