package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func roleVisibleToTenant(entity *model.RoleEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

// RoleSvc 平台排查视角：角色只读查看（列表/详情/成员）。
type RoleSvc interface {
	Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error)
	ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error)
}

type roleSvc struct{}

var _ RoleSvc = (*roleSvc)(nil)

func NewRoleSvc() RoleSvc {
	return &roleSvc{}
}

func (svc *roleSvc) Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if !roleVisibleToTenant(roleEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.RoleNotExistError)
	}

	resp := &dtopermission.RoleDetailResp{
		RoleID: roleEntity.ID,
		RoleBaseInfo: objpermission.RoleBaseInfo{
			TenantID:    roleEntity.TenantID,
			Name:        roleEntity.Name,
			Code:        roleEntity.Code,
			Description: roleEntity.Description,
			Type:        roleEntity.Type,
			IsDefault:   roleEntity.IsDefault,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: int64(roleEntity.CreatedAt.Unix()),
			UpdatedAt: int64(roleEntity.UpdatedAt.Unix()),
		},
	}
	return resp, nil
}

func (svc *roleSvc) PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error) {
	roleRepo := dao.NewRoleDao()
	cond := &dao.RoleCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
	}
	roleEntityList, total, err := roleRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListRole] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetPageListError)
	}

	list := make([]dtopermission.RolePageListItem, 0, len(roleEntityList))
	for _, v := range roleEntityList {
		list = append(list, dtopermission.RolePageListItem{
			RoleID: v.ID,
			RoleBaseInfo: objpermission.RoleBaseInfo{
				TenantID:    v.TenantID,
				Name:        v.Name,
				Code:        v.Code,
				Description: v.Description,
				Type:        v.Type,
				IsDefault:   v.IsDefault,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtopermission.RolePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *roleSvc) ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error) {
	userRoleDao := dao.NewUserRoleDao()
	userDao := dao.NewUserDao()

	list, err := userRoleDao.GetListByCond(ctx, &dao.UserRoleCond{
		RoleID: req.RoleID,
	})
	if err != nil {
		glog.Errorf(ctx, "[roleSvc.ListUsers] get users fail, err:%v", err)
		return nil, code.GetError(code.RoleUserGetListError)
	}

	userMap := make(map[string]*model.UserEntity)
	personMap := make(map[string]*model.PersonEntity)
	for _, ur := range list {
		if user, err := userDao.GetByID(ctx, ur.UserID); err == nil && user != nil {
			userMap[user.ID] = user
			if user.PersonID != "" {
				if person, perr := dao.NewPersonDao().GetByID(ctx, user.PersonID); perr == nil && person != nil && person.ID != "" {
					personMap[user.PersonID] = person
				}
			}
		}
	}

	users := make([]dtouser.RoleUserResp, 0, len(list))
	for _, ur := range list {
		if user, ok := userMap[ur.UserID]; ok {
			person := personMap[user.PersonID]
			if person == nil {
				person = &model.PersonEntity{}
			}
			users = append(users, dtouser.RoleUserResp{
				UserID:    ur.UserID,
				Username:  model.DerefStr(person.Username),
				Name:      user.Name,
				Email:     model.DerefStr(person.PrimaryEmail),
				RoleID:    ur.RoleID,
				CreatedAt: ur.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtouser.RoleUserListResp{
		Total: int64(len(users)),
		Users: users,
	}, nil
}
