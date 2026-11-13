// Package tenantadmin 是 tenantadmin 域的实现库（可被 sdk/server import）。
//
// 由原 iam/tenantadmin 的 DI 重构版迁入：controller/service/dto/router 全部扁平平铺
// 在本包之下（不再有 internal/）。对外通过 Mount 装配入口暴露：调用方（sdk/server）
// 持有 Domain→注册 key 的映射，把选中的 key 列表传进来，本库据此注册对应业务域。
//
// 依赖方向（分层倒置）：sdk/server → apps/tenantadmin（本库）→ pkg。
// 本库**不得** import sdk。
package tenantadmin

import (
	"github.com/gin-gonic/gin"
	tenantrouter "github.com/morehao/ark-iam/apps/tenantadmin/router"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// AppName 是 tenantadmin 应用的名称。
const AppName = "tenantadmin"

// Mount 按 keys（业务域注册 key）把对应业务域挂到 rg 上。
// keys 为空 = 挂载全部域。rg 应已带好 /v1/tenant 前缀（由调用方拼装）。
func Mount(rg *gin.RouterGroup, db gormdao.DBGetter, keys []string) {
	if len(keys) == 0 {
		for _, fn := range tenantrouter.Registry {
			fn(rg, db)
		}
		return
	}
	for _, key := range keys {
		fn, ok := tenantrouter.Registry[key]
		if !ok {
			continue
		}
		fn(rg, db)
	}
}
