package svcsession

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// SessionSvc 管理"我的会话"（即该 person 名下有效的 refresh token 记录）。
// 说明：会话列表与撤销操作的对象是 refresh_token 表中的记录（每个 refresh token
// 对应一个登录会话），而非 SSO 中心会话（Redis + session 审计表）。

type SessionSvc interface {
	List(ctx *gin.Context, req *dtouser.SessionListReq) (*dtouser.SessionListResp, error)
	Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq) error
	RevokeAll(ctx *gin.Context, req *dtouser.SessionRevokeAllReq) error
}

type sessionSvc struct{}

var _ SessionSvc = (*sessionSvc)(nil)

func NewSessionSvc() SessionSvc {
	return &sessionSvc{}
}

func (svc *sessionSvc) List(ctx *gin.Context, req *dtouser.SessionListReq) (*dtouser.SessionListResp, error) {
	refreshTokenDao := dao.NewRefreshTokenDao()
	personID := gincontext.GetPersonIDString(ctx)
	tenantID := gincontext.GetTenantIDString(ctx)

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	cond := &dao.RefreshTokenCond{
		BaseCond: &gormdao.BaseCond{Page: page, PageSize: pageSize},
		PersonID: personID,
		TenantID: tenantID,
	}
	list, total, err := refreshTokenDao.GetPageListByCond(ctx.Request.Context(), cond)
	if err != nil {
		glog.Errorf(ctx, "[sessionSvc.List] get page list fail, err:%v", err)
		return nil, code.GetError(code.SessionGetListError)
	}

	sessions := make([]dtouser.SessionResp, 0, len(list))
	now := time.Now()
	for _, item := range list {
		var isActive bool
		if item.RevokedAt != nil {
			isActive = false
		} else if item.ExpiredAt == nil || !item.ExpiredAt.After(now) {
			isActive = false
		} else {
			isActive = true
		}
		var expiresAt *int64
		if item.ExpiredAt != nil {
			t := item.ExpiredAt.Unix()
			expiresAt = &t
		}
		sessions = append(sessions, dtouser.SessionResp{
			// ID 即 refresh token 记录主键，也是撤销接口 :sessionID 路径参数的值
			ID:         item.ID,
			SessionID:  item.SessionID,
			AppID:      item.ApplicationClientID,
			TenantID:   item.TenantID,
			ClientType: item.ClientType,
			ClientIP:   item.ClientIP,
			UserAgent:  item.UserAgent,
			ExpiredAt:  expiresAt,
			CreatedAt:  item.CreatedAt.Unix(),
			IsActive:   isActive,
		})
	}

	return &dtouser.SessionListResp{
		List:  sessions,
		Total: total,
	}, nil
}

func (svc *sessionSvc) Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq) error {
	// 单条条件 UPDATE（id + person + tenant 归属），RowsAffected==0 即无权或不存在，
	// 替代"全量拉取 + 内存比对"的旧实现，消除归属校验与撤销之间的竞态。
	hit, err := dao.NewRefreshTokenDao().RevokeByID(ctx.Request.Context(), req.SessionID, gincontext.GetPersonIDString(ctx), gincontext.GetTenantIDString(ctx))
	if err != nil {
		glog.Errorf(ctx, "[sessionSvc.Revoke] revoke fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}
	if !hit {
		return code.GetError(code.SessionNotExistError)
	}
	return nil
}

func (svc *sessionSvc) RevokeAll(ctx *gin.Context, _ *dtouser.SessionRevokeAllReq) error {
	if err := dao.NewRefreshTokenDao().RevokeByCond(ctx.Request.Context(), &dao.RefreshTokenCond{
		PersonID: gincontext.GetPersonIDString(ctx),
		TenantID: gincontext.GetTenantIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[sessionSvc.RevokeAll] revoke all fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}
	return nil
}
