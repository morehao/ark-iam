package application

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
)

// AppIDByClientID 解析 client_id 所属应用——请求期入口（使用全局 iam 库）。
//
// 见 AppIDByClientIDWithDAO 的口径说明；本函数是「用全局库」的薄封装，
// 供请求期调用方（如 pkg/middleware.RequireAppAccess）使用。
func AppIDByClientID(ctx context.Context, clientID string) (string, error) {
	return AppIDByClientIDWithDAO(ctx, clientID, dao.NewApplicationClientDao())
}

// AppIDByClientIDWithDAO 解析本次签发客户端所属的应用——即 groups 声明的命名空间。
//
// client_id 全局唯一（`application_client.code` 带全局唯一索引），且解析发生在「租户确定之前」，
// 因此必须与 GetClientByClientID 同口径：显式声明跨全部租户读取。
//
// 返回空串表示「没有可用的命名空间」，此时调用方不产出声明而不报错——与「租户未定不猜租户」同构：
// clientID 为空（调用方拿不到客户端）或客户端未绑定应用，都属合法的「无作用域」。
// 客户端记录不存在则是状态异常，返回 error 由调用方 fail-closed 拒绝。
//
// clientDAO 是数据源注入点：协议层（oidcop）的持久化层按调用方事务/测试库装配 dao 并在此传入，
// 请求期调用方用 AppIDByClientID 走全局库。**解析逻辑只有本函数一份**，注入的只是数据源。
func AppIDByClientIDWithDAO(ctx context.Context, clientID string, clientDAO *dao.ApplicationClientDao) (string, error) {
	if clientID == "" {
		return "", nil
	}
	cctx := dbclient.CrossTenantContext(ctx)
	client, err := clientDAO.GetByCond(cctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil {
		return "", err
	}
	if client == nil || client.ID == "" {
		return "", fmt.Errorf("application client not found: %s", clientID)
	}
	return client.AppID, nil
}
