// Package application 提供跨应用复用的「应用入口策略」领域能力：
//   - 按 OIDC client_id（= application_client.code）解析其归属应用；
//   - 语义化读取应用级入口策略开关。
//
// 两个开关分别门禁两条自助通道，且**都是应用级策略、不是全局开关**：
//   - AllowPersonCreateTenant：通道 A（POST /oidc/registerPerson → POST /oidc/createTenant）；
//   - AllowJoinByInvite：通道 B（POST /v1/auth/joinTenant 凭邀请加入已有租户）。
//
// 两端判定都必须"先按 client_id 找到应用、再读该应用的开关"，避免各自实现一套解析逻辑
// 造成同一字段两处口径（历史上 AllowJoinByInvite 就是因为没有共用实现而长期零消费）。
//
// 本包只依赖 pkg/{dao,model}，不引用任何具体应用。
package application

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/glog"
)

// GetByClientID 按 OIDC client_id 解析其归属应用。
//
// clientID 为空、客户端不存在或应用不存在时返回 (nil, nil)——「解析不出应用」是可预期
// 的业务边界，由调用方决定如何处理（各门禁一律 fail-closed）；仅返回数据库/IO 等系统错误。
func GetByClientID(ctx *gin.Context, clientID string) (*model.ApplicationEntity, error) {
	if clientID == "" {
		return nil, nil
	}
	// client_id 全局唯一，且本函数在"租户确定之前"的协议层入口被调用（注册/加入租户通道），
	// 因此必须显式声明「全租户」作用域：跨租户可见性只能来自显式声明，不能来自缺失作用域。
	clientEntity, err := dao.NewApplicationClientDao().GetByCond(dbclient.CrossTenantContext(ctx), &dao.ApplicationClientCond{Code: clientID})
	if err != nil {
		glog.Errorf(ctx, "[application.GetByClientID] dao applicationClient GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, err
	}
	if clientEntity == nil || clientEntity.ID == "" || clientEntity.AppID == "" {
		return nil, nil
	}
	appEntity, err := dao.NewApplicationDao().GetByID(ctx, clientEntity.AppID)
	if err != nil {
		glog.Errorf(ctx, "[application.GetByClientID] dao application GetByID fail, err:%v, appID:%s", err, clientEntity.AppID)
		return nil, err
	}
	if appEntity == nil || appEntity.ID == "" {
		return nil, nil
	}
	return appEntity, nil
}

// AllowsPersonCreateTenant 应用是否允许其用户自助开通租户（通道 A）。
// 字段可空（NULL = 未配置），未配置视为不允许。
func AllowsPersonCreateTenant(app *model.ApplicationEntity) bool {
	return boolPtrValue(app, func(a *model.ApplicationEntity) *bool { return a.AllowPersonCreateTenant })
}

// AllowsJoinByInvite 应用是否允许其用户凭邀请加入已有租户（通道 B）。
// 字段可空（NULL = 未配置），未配置视为不允许。
func AllowsJoinByInvite(app *model.ApplicationEntity) bool {
	return boolPtrValue(app, func(a *model.ApplicationEntity) *bool { return a.AllowJoinByInvite })
}

func boolPtrValue(app *model.ApplicationEntity, pick func(*model.ApplicationEntity) *bool) bool {
	if app == nil {
		return false
	}
	v := pick(app)
	return v != nil && *v
}
