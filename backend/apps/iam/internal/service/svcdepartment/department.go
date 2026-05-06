package svcdepartment

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtodepartment"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objdepartment"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type DepartmentSvc interface {
	Create(ctx *gin.Context, req *dtodepartment.DepartmentCreateReq) (*dtodepartment.DepartmentCreateResp, error)
	Delete(ctx *gin.Context, req *dtodepartment.DepartmentDeleteReq) error
	Update(ctx *gin.Context, req *dtodepartment.DepartmentUpdateReq) error
	Detail(ctx *gin.Context, req *dtodepartment.DepartmentDetailReq) (*dtodepartment.DepartmentDetailResp, error)
	PageList(ctx *gin.Context, req *dtodepartment.DepartmentPageListReq) (*dtodepartment.DepartmentPageListResp, error)
	Tree(ctx *gin.Context, req *dtodepartment.DepartmentTreeReq) (*dtodepartment.DepartmentTreeResp, error)
}

type departmentSvc struct {
}

var _ DepartmentSvc = (*departmentSvc)(nil)

func NewDepartmentSvc() DepartmentSvc {
	return &departmentSvc{}
}

func (svc *departmentSvc) Create(ctx *gin.Context, req *dtodepartment.DepartmentCreateReq) (*dtodepartment.DepartmentCreateResp, error) {
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
		glog.Errorf(ctx, "[svcdepartment.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentCreateError)
	}
	return &dtodepartment.DepartmentCreateResp{
		DepartmentID: insertEntity.ID,
	}, nil
}

func (svc *departmentSvc) Delete(ctx *gin.Context, req *dtodepartment.DepartmentDeleteReq) error {
	departmentEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	if departmentEntity == nil || departmentEntity.ID == 0 {
		return code.GetError(code.DepartmentNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewDepartmentDao().Delete(ctx, req.DepartmentID, userID); err != nil {
		glog.Errorf(ctx, "[svcdepartment.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentDeleteError)
	}
	return nil
}

func (svc *departmentSvc) Update(ctx *gin.Context, req *dtodepartment.DepartmentUpdateReq) error {
	departmentEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	if departmentEntity == nil || departmentEntity.ID == 0 {
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
		glog.Errorf(ctx, "[svcdepartment.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DepartmentUpdateError)
	}
	return nil
}

func (svc *departmentSvc) Detail(ctx *gin.Context, req *dtodepartment.DepartmentDetailReq) (*dtodepartment.DepartmentDetailResp, error) {
	departmentEntity, err := dao.NewDepartmentDao().GetByID(ctx, req.DepartmentID)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetDetailError)
	}
	if departmentEntity == nil || departmentEntity.ID == 0 {
		return nil, code.GetError(code.DepartmentNotExistError)
	}

	resp := &dtodepartment.DepartmentDetailResp{
		DepartmentID: departmentEntity.ID,
		DepartmentBaseInfo: objdepartment.DepartmentBaseInfo{
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

func (svc *departmentSvc) PageList(ctx *gin.Context, req *dtodepartment.DepartmentPageListReq) (*dtodepartment.DepartmentPageListResp, error) {
	cond := &dao.DepartmentCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Code:     req.Code,
	}
	departmentEntityList, total, err := dao.NewDepartmentDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	list := make([]dtodepartment.DepartmentPageListItem, 0, len(departmentEntityList))
	for _, v := range departmentEntityList {
		list = append(list, dtodepartment.DepartmentPageListItem{
			DepartmentID: v.ID,
			DepartmentBaseInfo: objdepartment.DepartmentBaseInfo{
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
	return &dtodepartment.DepartmentPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *departmentSvc) Tree(ctx *gin.Context, req *dtodepartment.DepartmentTreeReq) (*dtodepartment.DepartmentTreeResp, error) {
	cond := &dao.DepartmentCond{
		TenantID: req.TenantID,
	}
	departmentEntityList, _, err := dao.NewDepartmentDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdepartment.Tree] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DepartmentGetPageListError)
	}

	deptMap := make(map[uint]*model.DepartmentEntity)
	for i := range departmentEntityList {
		deptMap[departmentEntityList[i].ID] = &departmentEntityList[i]
	}

	var buildTree func(parentID uint) []dtodepartment.DepartmentTreeItem
	buildTree = func(parentID uint) []dtodepartment.DepartmentTreeItem {
		var items []dtodepartment.DepartmentTreeItem
		for _, dept := range departmentEntityList {
			if dept.ParentID == parentID {
				item := dtodepartment.DepartmentTreeItem{
					DepartmentID: dept.ID,
					DepartmentBaseInfo: objdepartment.DepartmentBaseInfo{
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

	return &dtodepartment.DepartmentTreeResp{
		List: buildTree(0),
	}, nil
}