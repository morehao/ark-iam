package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// OrganizationSvc 平台侧组织只读视图（跨租户排查用，无写操作）。
type OrganizationSvc interface {
	Tree(ctx *gin.Context, req *dtotenant.OrganizationTreeReq) (*dtotenant.OrganizationTreeResp, error)
	GetUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationListReq) (*dtotenant.UserOrganizationListResp, error)
}

type organizationSvc struct {
}

var _ OrganizationSvc = (*organizationSvc)(nil)

func NewOrganizationSvc() OrganizationSvc {
	return &organizationSvc{}
}

// Tree 按租户只读组织树（tenantID 必填）。
func (svc *organizationSvc) Tree(ctx *gin.Context, req *dtotenant.OrganizationTreeReq) (*dtotenant.OrganizationTreeResp, error) {
	if req.TenantID == "" {
		return nil, code.GetError(code.OrganizationGetPageListError)
	}
	orgEntityList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{
		TenantID: req.TenantID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.OrganizationTree] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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

// GetUserOrganizations 用户组织归属只读查询（含组织名）。
func (svc *organizationSvc) GetUserOrganizations(ctx *gin.Context, req *dtotenant.UserOrganizationListReq) (*dtotenant.UserOrganizationListResp, error) {
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		UserID: req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.GetUserOrganizations] dao GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationUserGetPageListError)
	}
	orgNameMap := make(map[string]string)
	for _, r := range relationList {
		if _, ok := orgNameMap[r.OrganizationID]; ok {
			continue
		}
		if o, err := dao.NewOrganizationDao().GetByID(ctx, r.OrganizationID); err == nil && o != nil {
			orgNameMap[r.OrganizationID] = o.Name
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
