package svcsession

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/glog"
)

type SessionSvc interface {
	List(ctx *gin.Context, req *dtouser.SessionListReq, userID uint) (*dtouser.SessionListResp, error)
	Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID uint) error
	RevokeAll(ctx *gin.Context, userID uint) error
}

type sessionSvc struct{}

var _ SessionSvc = (*sessionSvc)(nil)

func NewSessionSvc() SessionSvc {
	return &sessionSvc{}
}

func (svc *sessionSvc) List(ctx *gin.Context, req *dtouser.SessionListReq, userID uint) (*dtouser.SessionListResp, error) {
	sessionDao := dao.NewSessionDao()

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	list, total, err := sessionDao.GetPageListByCond(ctx.Request.Context(), &dao.SessionCond{
		UserID: userID,
	}, page, pageSize)
	if err != nil {
		glog.Errorf(ctx, "[sessionSvc.List] get page list fail, err:%v", err)
		return nil, code.GetError(code.SessionGetListError)
	}

	sessions := make([]dtouser.SessionResp, 0, len(list))
	now := time.Now()
	for _, item := range list {
		var isActive bool
		if item.RevokedAt.Valid {
			isActive = false
		} else if item.ExpiresAt.Valid && item.ExpiresAt.Time.Before(now) {
			isActive = false
		} else {
			isActive = true
		}
		expiresAt := ""
		if item.ExpiresAt.Valid {
			expiresAt = item.ExpiresAt.Time.Format("2006-01-02 15:04:05")
		}
		sessions = append(sessions, dtouser.SessionResp{
			ID:            uint64(item.ID),
			ApplicationID: uint64(item.ApplicationID),
			TenantID:      uint64(item.TenantID),
			ExpiresAt:     &expiresAt,
			CreatedAt:     item.CreatedAt.Format("2006-01-02 15:04:05"),
			IsActive:      isActive,
		})
	}

	return &dtouser.SessionListResp{
		List:  sessions,
		Total: total,
	}, nil
}

func (svc *sessionSvc) Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID uint) error {
	sessionDao := dao.NewSessionDao()

	if err := sessionDao.RevokeByID(ctx.Request.Context(), uint(req.SessionID), userID); err != nil {
		glog.Errorf(ctx, "[sessionSvc.Revoke] revoke fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}

func (svc *sessionSvc) RevokeAll(ctx *gin.Context, userID uint) error {
	sessionDao := dao.NewSessionDao()

	if err := sessionDao.RevokeAllByUserID(ctx, userID); err != nil {
		glog.Errorf(ctx, "[sessionSvc.RevokeAll] revoke all fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}