package svcsession

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

type SessionSvc interface {
	List(ctx *gin.Context, req *dtouser.SessionListReq, personID, userID, tenantID uint) (*dtouser.SessionListResp, error)
	Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID, tenantID, personID uint) error
	RevokeAll(ctx *gin.Context, userID, tenantID, personID uint) error
}

type sessionStore interface {
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.RefreshTokenEntityList, int64, error)
	UpdateMap(ctx context.Context, id uint, updates map[string]any) error
}

var newSessionStore = func() sessionStore {
	return dao.NewSessionDao()
}

type sessionSvc struct{}

var _ SessionSvc = (*sessionSvc)(nil)

func NewSessionSvc() SessionSvc {
	return &sessionSvc{}
}

func (svc *sessionSvc) List(ctx *gin.Context, req *dtouser.SessionListReq, personID, userID, tenantID uint) (*dtouser.SessionListResp, error) {
	sessionDao := newSessionStore()

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	cond := &dao.SessionCond{
		BaseCond: &gormdao.BaseCond{Page: page, PageSize: pageSize},
		PersonID: personID,
		UserID:   userID,
		TenantID: tenantID,
	}
	list, total, err := sessionDao.GetPageListByCond(ctx.Request.Context(), cond)
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
		expiresAt := ""
		if item.ExpiredAt != nil {
			expiresAt = item.ExpiredAt.Format("2006-01-02 15:04:05")
		}
		sessions = append(sessions, dtouser.SessionResp{
			ID:         uint64(item.ID),
			SessionID:  item.SessionID,
			AppID:      uint64(item.ApplicationClientID),
			TenantID:   uint64(item.TenantID),
			ClientType: item.ClientType,
			ClientIP:   item.ClientIP,
			UserAgent:  item.UserAgent,
			ExpiredAt:  &expiresAt,
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
			IsActive:   isActive,
		})
	}

	return &dtouser.SessionListResp{
		List:  sessions,
		Total: total,
	}, nil
}

func (svc *sessionSvc) Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID, tenantID, personID uint) error {
	sessionDao := newSessionStore()

	if err := sessionDao.UpdateMap(ctx.Request.Context(), uint(req.SessionID), map[string]any{"revoked_at": gorm.Expr("NOW()")}); err != nil {
		glog.Errorf(ctx, "[sessionSvc.Revoke] revoke fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}

func (svc *sessionSvc) RevokeAll(ctx *gin.Context, userID, tenantID, personID uint) error {
	cond := &dao.SessionCond{
		PersonID: personID,
		TenantID: tenantID,
		UserID:   userID,
	}
	if err := cond.RevokeAll(ctx.Request.Context()); err != nil {
		glog.Errorf(ctx, "[sessionSvc.RevokeAll] revoke all fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}
