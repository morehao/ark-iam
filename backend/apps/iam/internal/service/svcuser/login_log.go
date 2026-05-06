package svcuser

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func (svc *userSvc) DetailUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogDetailReq) (*dtouser.UserLoginLogDetailResp, error) {
	userLoginLogEntity, err := dao.NewUserLoginLogDao().GetByID(ctx, req.UserLoginLogID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DetailUserLoginLog] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetDetailError)
	}
	if userLoginLogEntity == nil || userLoginLogEntity.ID == 0 {
		return nil, code.GetError(code.UserLoginLogNotExistError)
	}

	resp := &dtouser.UserLoginLogDetailResp{
		UserLoginLogID: userLoginLogEntity.ID,
		TenantID:       userLoginLogEntity.TenantID,
		UserID:         userLoginLogEntity.UserID,
		LoginIP:        userLoginLogEntity.LoginIP,
		UserAgent:      userLoginLogEntity.UserAgent,
		LoginTime:      userLoginLogEntity.LoginTime.Unix(),
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userLoginLogEntity.CreatedAt.Unix(),
			UpdatedAt: userLoginLogEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageListUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogPageListReq) (*dtouser.UserLoginLogPageListResp, error) {
	cond := &dao.UserLoginLogCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		UserID:   req.UserID,
		LoginIP:  req.LoginIP,
	}
	userLoginLogEntityList, total, err := dao.NewUserLoginLogDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageListUserLoginLog] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetPageListError)
	}

	list := make([]dtouser.UserLoginLogPageListItem, 0, len(userLoginLogEntityList))
	for _, v := range userLoginLogEntityList {
		list = append(list, dtouser.UserLoginLogPageListItem{
			UserLoginLogID: v.ID,
			TenantID:       v.TenantID,
			UserID:         v.UserID,
			LoginIP:        v.LoginIP,
			UserAgent:      v.UserAgent,
			LoginTime:      v.LoginTime.Unix(),
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserLoginLogPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *userSvc) GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error) {
	cond := &dao.UserLoginLogCond{
		UserID: req.UserID,
	}
	userLoginLogEntityList, total, err := dao.NewUserLoginLogDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.GetUserLoginLogByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetPageListError)
	}

	list := make([]dtouser.UserLoginLogPageListItem, 0, len(userLoginLogEntityList))
	for _, v := range userLoginLogEntityList {
		list = append(list, dtouser.UserLoginLogPageListItem{
			UserLoginLogID: v.ID,
			TenantID:       v.TenantID,
			UserID:         v.UserID,
			LoginIP:        v.LoginIP,
			UserAgent:      v.UserAgent,
			LoginTime:      v.LoginTime.Unix(),
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserLoginLogPageListResp{
		List:  list,
		Total: total,
	}, nil
}
