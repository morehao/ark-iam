package svctenant

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type UserSvc interface {
	PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// PageList 返回当前租户内的用户目录（含自然人基础信息），
// 供租户自服务端（如组织用户/组织角色用户的选择器）使用。
func (svc *userSvc) PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error) {
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
	}
	userEntityList, total, err := dao.NewUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}

	personIDs := make([]uint, 0, len(userEntityList))
	for _, v := range userEntityList {
		if v.PersonID != 0 {
			personIDs = append(personIDs, v.PersonID)
		}
	}
	personMap := loadPersonMap(ctx, personIDs)

	list := make([]dtotenant.UserPageListItem, 0, len(userEntityList))
	for _, v := range userEntityList {
		person := personMap[v.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		list = append(list, dtotenant.UserPageListItem{
			UserID:       v.ID,
			TenantID:     v.TenantID,
			Username:     model.DerefStr(person.Username),
			PrimaryEmail: model.DerefStr(person.PrimaryEmail),
			PrimaryPhone: model.DerefStr(person.PrimaryPhone),
			Name:         v.Name,
			Avatar:       v.Avatar,
			IsSuspended:  v.IsSuspended,
			CreatedAt:    v.CreatedAt.Unix(),
		})
	}
	return &dtotenant.UserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// loadPersonMap 批量加载自然人信息（username/email/phone 来自 person 表）。
func loadPersonMap(ctx context.Context, personIDs []uint) map[uint]*model.PersonEntity {
	result := make(map[uint]*model.PersonEntity)
	if len(personIDs) == 0 {
		return result
	}
	personDao := dao.NewPersonDao()
	for _, id := range personIDs {
		if id == 0 {
			continue
		}
		person, err := personDao.GetByID(ctx, id)
		if err != nil {
			glog.Warnf(ctx, "[svcuser.loadPersonMap] person GetByID fail, personID:%d, err:%v", id, err)
			continue
		}
		if person != nil && person.ID != 0 {
			result[id] = person
		}
	}
	return result
}
