package svctenant

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// InviteSvc 租户邀请管理：租户 owner/管理员生成加入邀请，凭证持有者凭邀请码自助加入租户（通道 B）。
type InviteSvc interface {
	Create(ctx *gin.Context, req *dtotenant.InviteCreateReq) (*dtotenant.InviteCreateResp, error)
	Revoke(ctx *gin.Context, req *dtotenant.InviteRevokeReq) error
	PageList(ctx *gin.Context, req *dtotenant.InvitePageListReq) (*dtotenant.InvitePageListResp, error)
}

type inviteSvc struct{}

var _ InviteSvc = (*inviteSvc)(nil)

func NewInviteSvc() InviteSvc {
	return &inviteSvc{}
}

// Create 生成一条租户加入邀请（一次性邀请码，可选有效期）。
func (svc *inviteSvc) Create(ctx *gin.Context, req *dtotenant.InviteCreateReq) (*dtotenant.InviteCreateResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	operatorID := gincontext.GetUserIDString(ctx)

	var expiresAt *time.Time
	if req.ExpireHours > 0 {
		t := time.Now().Add(time.Duration(req.ExpireHours) * time.Hour)
		expiresAt = &t
	}

	entity := &model.InviteEntity{
		TenantID:  tenantID,
		Code:      "invite-" + uuid.NewString(),
		Status:    model.InviteStatusPending,
		ExpiresAt: expiresAt,
		CreatedBy: operatorID,
	}
	if err := dao.NewInviteDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcinvite.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.InviteCreateError)
	}
	return &dtotenant.InviteCreateResp{
		InviteID: entity.ID,
		Code:     entity.Code,
	}, nil
}

// Revoke 撤销一条邀请（置为 revoked 状态）。
func (svc *inviteSvc) Revoke(ctx *gin.Context, req *dtotenant.InviteRevokeReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	entity, err := dao.NewInviteDao().GetByID(ctx, req.InviteID)
	if err != nil {
		glog.Errorf(ctx, "[svcinvite.Revoke] dao GetByID fail, err:%v, inviteID:%s", err, req.InviteID)
		return code.GetError(code.InviteRevokeError)
	}
	if entity == nil || entity.ID == "" || entity.TenantID != tenantID {
		return code.GetError(code.InviteInvalidError)
	}
	if err := dao.NewInviteDao().UpdateMap(ctx, req.InviteID, map[string]any{
		"status":     string(model.InviteStatusRevoked),
		"updated_by": gincontext.GetUserIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcinvite.Revoke] dao UpdateMap fail, err:%v, inviteID:%s", err, req.InviteID)
		return code.GetError(code.InviteRevokeError)
	}
	return nil
}

// PageList 邀请列表（租户内）。
func (svc *inviteSvc) PageList(ctx *gin.Context, req *dtotenant.InvitePageListReq) (*dtotenant.InvitePageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.InviteCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		Status:   req.Status,
	}
	list, total, err := dao.NewInviteDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcinvite.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.InviteGetPageListError)
	}
	items := make([]dtotenant.InvitePageListItem, 0, len(list))
	for _, v := range list {
		var exp *int64
		if v.ExpiresAt != nil {
			e := v.ExpiresAt.Unix()
			exp = &e
		}
		items = append(items, dtotenant.InvitePageListItem{
			InviteID:  v.ID,
			Code:      v.Code,
			Status:    string(v.Status),
			ExpiresAt: exp,
			CreatedAt: v.CreatedAt.Unix(),
		})
	}
	return &dtotenant.InvitePageListResp{
		List:  items,
		Total: total,
	}, nil
}
