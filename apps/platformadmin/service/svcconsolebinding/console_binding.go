package svcconsolebinding

import (
	"context"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoconsolebinding"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// ConsoleBindingSvc 平台管理员维护 Host → 租户/issuer/client 的 console binding。
// P0 只开放 enable/disable 动作与列表；删除走 offboard/release（自动 disable+软删）。
type ConsoleBindingSvc interface {
	List(ctx context.Context, req *dtoconsolebinding.ConsoleBindingListReq) (*dtoconsolebinding.ConsoleBindingListResp, error)
	Enable(ctx context.Context, req *dtoconsolebinding.ConsoleBindingEnableReq) error
	Disable(ctx context.Context, req *dtoconsolebinding.ConsoleBindingDisableReq) error
}

type consoleBindingSvc struct {
	db gormdao.DBGetter
}

var _ ConsoleBindingSvc = (*consoleBindingSvc)(nil)

func NewConsoleBindingSvc(db gormdao.DBGetter) ConsoleBindingSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &consoleBindingSvc{db: db}
}

func (svc *consoleBindingSvc) List(ctx context.Context, req *dtoconsolebinding.ConsoleBindingListReq) (*dtoconsolebinding.ConsoleBindingListResp, error) {
	cond := &dao.ConsoleBindingCond{
		BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		Host:     req.Host, TenantID: req.TenantID, Kind: req.Kind, Status: req.Status,
	}
	list, err := dao.NewConsoleBindingDao(dao.WithDBGetter(svc.db)).List(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcconsolebinding.List] fail, err:%v", err)
		return nil, code.GetError(code.DomainGetPageListError)
	}
	items := make([]dtoconsolebinding.ConsoleBindingItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoconsolebinding.ConsoleBindingItem{
			ID: v.ID, Host: v.Host, TenantID: v.TenantID, Issuer: v.Issuer, ClientID: v.ClientID,
			Audience: v.Audience, RedirectURI: v.RedirectURI, Kind: v.Kind, Status: v.Status,
			CreatedAt: v.CreatedAt.Unix(), UpdatedAt: v.UpdatedAt.Unix(),
		})
	}
	return &dtoconsolebinding.ConsoleBindingListResp{List: items, Total: int64(len(items))}, nil
}

// Enable 启用 binding：要求 primary active + D console client 有效 + tenant/issuer/client 一致（DG-ONBOARD-1）。
func (svc *consoleBindingSvc) Enable(ctx context.Context, req *dtoconsolebinding.ConsoleBindingEnableReq) error {
	b, err := svc.load(ctx, req.BindingID)
	if err != nil {
		return err
	}
	if b.Status == model.ConsoleBindingStatusEnable {
		return nil
	}
	// 校验：binding 的 tenant/issuer/client 三者一致，且 tenant 已 active、issue 非空。
	var tn model.TenantEntity
	if err := svc.db(ctx).WithContext(dbclient.CrossTenantContext(ctx)).
		Where("id = ? AND deleted_at IS NULL", b.TenantID).First(&tn).Error; err != nil {
		glog.Errorf(ctx, "[svcconsolebinding.Enable] load tenant fail, err:%v", err)
		return code.GetError(code.DomainTenantSuspendedError)
	}
	if tn.Status != model.TenantStatusActive {
		glog.Errorf(ctx, "[svcconsolebinding.Enable] tenant not active, tenantID:%s, status:%s", b.TenantID, tn.Status)
		return code.GetError(code.DomainTenantSuspendedError)
	}
	if b.Issuer == "" || b.Issuer != tn.Issuer {
		glog.Errorf(ctx, "[svcconsolebinding.Enable] binding issuer mismatch, binding=%q tenant=%q", b.Issuer, tn.Issuer)
		return code.GetError(code.DomainIssuerTierMismatchError)
	}
	if b.ClientID == "" {
		glog.Errorf(ctx, "[svcconsolebinding.Enable] binding client_id empty, bindingID:%s", b.ID)
		return code.GetError(code.DomainIssuerTierMismatchError)
	}
	if err := dao.NewConsoleBindingDao(dao.WithDBGetter(svc.db)).UpdateStatus(ctx, nil, b.ID, b.TenantID, model.ConsoleBindingStatusEnable, req.Operator.UserID); err != nil {
		glog.Errorf(ctx, "[svcconsolebinding.Enable] update fail, err:%v", err)
		return code.GetError(code.DomainUpdateError)
	}
	return nil
}

// Disable 停用 binding（可恢复；offboard/release 自动调用）。
func (svc *consoleBindingSvc) Disable(ctx context.Context, req *dtoconsolebinding.ConsoleBindingDisableReq) error {
	b, err := svc.load(ctx, req.BindingID)
	if err != nil {
		return err
	}
	if err := dao.NewConsoleBindingDao(dao.WithDBGetter(svc.db)).UpdateStatus(ctx, nil, b.ID, b.TenantID, model.ConsoleBindingStatusDisable, req.Operator.UserID); err != nil {
		glog.Errorf(ctx, "[svcconsolebinding.Disable] update fail, err:%v", err)
		return code.GetError(code.DomainUpdateError)
	}
	return nil
}

func (svc *consoleBindingSvc) load(ctx context.Context, bindingID string) (*model.ConsoleBindingEntity, error) {
	b, err := dao.NewConsoleBindingDao(dao.WithDBGetter(svc.db)).GetByID(dbclient.CrossTenantContext(ctx), bindingID)
	if err != nil {
		glog.Errorf(ctx, "[svcconsolebinding.load] GetByID fail, err:%v", err)
		return nil, code.GetError(code.DomainDetailError)
	}
	if b == nil {
		return nil, code.GetError(code.DomainNotExistError)
	}
	return b, nil
}
