package svcoidc

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
)

// ChangePassword 首次登录强制改密（临时密码场景，见
// docs/design/tenant-admin-provisioning-design-20260912.md D3）。
//
// 前置：CompleteLogin 命中 person.must_change_password=true 时不完成授权（done=false），
// 仅把 subject 绑到授权票据并返回 RequiresPasswordChange，前端据此引导用户设置新密码。
//
// 本接口只做"改密"这一件事：由授权票据的 subject 定位自然人（不接受请求体传 personID，
// 避免参数污染）→ 校验当前（临时）密码 → 强度与同值校验 → 更新口令、清除强制改密标记 →
// 撤销既有 SSO 会话与 refresh token。改密后不续跑登录流程——租户选择与应用策略需要在
// 重新登录时按最新状态重新判定，前端回到登录页用新密码重新登录。
func (svc *oidcAuthSvc) ChangePassword(ctx *gin.Context, req *dtooidc.OIDCChangePasswordReq) error {
	reqCtx := ctx.Request.Context()
	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, req.AuthRequestID)
	if err != nil {
		return mapAuthRequestError(err)
	}
	personID, perr := oidcop.ParseSubject(authReq.GetSubject())
	if perr != nil {
		return code.GetError(code.OIDCSessionNotFound)
	}

	personDao := dao.NewPersonDao()
	personEntity, err := personDao.GetByID(reqCtx, personID)
	if err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.ChangePassword] dao GetByID person fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == "" {
		return code.GetError(code.OIDCSessionNotFound)
	}
	// 只服务"临时密码首次登录"场景：非强制改密的改密走 /v1/auth/me/password，
	// 此处收紧可减少一条仅凭授权票据即可改密的入口。
	if !personEntity.MustChangePassword {
		return code.GetError(code.OIDCSessionNotFound)
	}
	// 连接器注册的账号可能没有本地密码，此时提示"密码未设置"而非"密码错误"
	if personEntity.PasswordEncrypted == "" {
		return code.GetError(code.PasswordNotSetError)
	}
	if err := credential.ValidateStrength(req.NewPassword); err != nil {
		return code.GetError(code.PasswordValidationError)
	}
	if req.CurrentPassword == req.NewPassword {
		glog.Warnf(ctx, "[oidcAuthSvc.ChangePassword] new password equals current password, personID:%s", personID)
		return code.GetError(code.PasswordValidationError)
	}
	if err := gcrypto.ComparePasswordHash(personEntity.PasswordEncrypted, req.CurrentPassword); err != nil {
		glog.Warnf(ctx, "[oidcAuthSvc.ChangePassword] current password mismatch, personID:%s", personID)
		return code.GetError(code.PasswordMismatchError)
	}
	newHash, err := gcrypto.GeneratePasswordHash(req.NewPassword)
	if err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.ChangePassword] GeneratePasswordHash fail, err:%v", err)
		return code.GetError(code.PasswordHashError)
	}
	if err := personDao.UpdateMap(reqCtx, personID, map[string]any{
		"password_encrypted":   newHash,
		"password_method":      model.PasswordMethodBcrypt,
		"must_change_password": false,
		"updated_by":           personID,
	}); err != nil {
		glog.Errorf(ctx, "[oidcAuthSvc.ChangePassword] dao UpdateMap fail, err:%v, personID:%s", err, personID)
		return code.GetError(code.UserUpdateError)
	}

	// 改密即全局登出：临时密码登录过程中可能已建立的会话在改密后立即失效
	tenant.RevokePersonSessions(reqCtx, personID)
	return nil
}
