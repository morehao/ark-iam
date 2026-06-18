package svcdomain

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtodomain"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type domainRepository interface {
	Insert(ctx context.Context, entity *model.DomainEntity) error
	GetByID(ctx context.Context, id uint) (*model.DomainEntity, error)
	GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.DomainEntityList, int64, error)
	GetByTenantAndDomain(ctx context.Context, tenantID uint, domain string) (*model.DomainEntity, error)
	UpdateMap(ctx context.Context, id uint, updateMap map[string]any) error
	Delete(ctx context.Context, id uint, deletedBy uint) error
}

var newDomainRepo = func() domainRepository {
	return dao.NewDomainDao()
}

type DomainSvc interface {
	Create(ctx *gin.Context, req *dtodomain.CreateDomainReq) (*dtodomain.DomainCreateResp, error)
	Update(ctx *gin.Context, req *dtodomain.UpdateDomainReq) error
	Detail(ctx *gin.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error)
	PageList(ctx *gin.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error)
	Delete(ctx *gin.Context, req *dtodomain.DeleteDomainReq) error
}

type domainSvc struct{}

var _ DomainSvc = (*domainSvc)(nil)

func NewDomainSvc() DomainSvc {
	return &domainSvc{}
}

func (svc *domainSvc) Create(ctx *gin.Context, req *dtodomain.CreateDomainReq) (*dtodomain.DomainCreateResp, error) {
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return nil, code.GetError(code.DomainCreateError)
	}
	if len(domain) > 256 {
		return nil, code.GetError(code.DomainCreateError)
	}

	tenantID := gincontext.GetTenantID(ctx)

	repo := newDomainRepo()
	existing, err := repo.GetByTenantAndDomain(ctx, tenantID, domain)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Create] GetByTenantAndDomain fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainCreateError)
	}
	if existing != nil {
		return nil, code.GetError(code.DomainAlreadyExistError)
	}

	entity := &model.DomainEntity{
		TenantID:   tenantID,
		Domain:     domain,
		IsVerified: 0,
		CreatedBy:  gincontext.GetUserID(ctx),
		UpdatedBy:  gincontext.GetUserID(ctx),
	}
	if err := repo.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcdomain.Create] Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainCreateError)
	}
	return &dtodomain.DomainCreateResp{ID: entity.ID}, nil
}

func (svc *domainSvc) PageList(ctx *gin.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error) {
	tenantID := gincontext.GetTenantID(ctx)

	cond := &dao.DomainCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		Domain:   req.Domain,
	}

	repo := newDomainRepo()
	list, total, err := repo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.PageList] GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainGetPageListError)
	}

	items := make([]dtodomain.DomainPageListItem, 0, len(list))
	for _, v := range list {
		verifiedAt := ""
		if v.VerifiedAt.Valid {
			verifiedAt = v.VerifiedAt.Time.Format("2006-01-02 15:04:05")
		}
		items = append(items, dtodomain.DomainPageListItem{
			ID:         v.ID,
			Domain:     v.Domain,
			IsVerified: v.IsVerified,
			VerifiedAt: verifiedAt,
			CreatedAt:  v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtodomain.DomainPageListResp{List: items, Total: total}, nil
}

func (svc *domainSvc) Delete(ctx *gin.Context, req *dtodomain.DeleteDomainReq) error {
	tenantID := gincontext.GetTenantID(ctx)

	repo := newDomainRepo()
	entity, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Delete] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainDeleteError)
	}
	if entity == nil || entity.ID == 0 || entity.TenantID != tenantID {
		return code.GetError(code.DomainNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := repo.Delete(ctx, req.ID, userID); err != nil {
		glog.Errorf(ctx, "[svcdomain.Delete] Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainDeleteError)
	}
	return nil
}

func (svc *domainSvc) Update(ctx *gin.Context, req *dtodomain.UpdateDomainReq) error {
	tenantID := gincontext.GetTenantID(ctx)

	repo := newDomainRepo()
	entity, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Update] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainUpdateError)
	}
	if entity == nil || entity.ID == 0 || entity.TenantID != tenantID {
		return code.GetError(code.DomainNotExistError)
	}

	updateMap := map[string]any{
		"updated_by": gincontext.GetUserID(ctx),
	}
	if req.Domain != "" {
		updateMap["domain"] = req.Domain
	}
	if req.IsVerified != nil {
		updateMap["is_verified"] = *req.IsVerified
	}

	if err := repo.UpdateMap(ctx, req.ID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcdomain.Update] UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainUpdateError)
	}
	return nil
}

func (svc *domainSvc) Detail(ctx *gin.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error) {
	tenantID := gincontext.GetTenantID(ctx)

	repo := newDomainRepo()
	entity, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Detail] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainDetailError)
	}
	if entity == nil || entity.ID == 0 || entity.TenantID != tenantID {
		return nil, code.GetError(code.DomainNotExistError)
	}

	verifiedAt := ""
	if entity.VerifiedAt.Valid {
		verifiedAt = entity.VerifiedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &dtodomain.DomainDetailResp{
		ID:         entity.ID,
		Domain:     entity.Domain,
		IsVerified: entity.IsVerified,
		VerifiedAt: verifiedAt,
		CreatedAt:  entity.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  entity.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
