// Package rpapi 装配「面向应用的只读目录 API」应用。
//
// 职责单一：只读 /v1/rp/directory/*，按令牌的租户作用域返回本租户目录数据。
// 与 OP（auth）在受众与爆炸半径上隔离：目录 API 的故障或流量高峰不影响登录。
package rpapi

import (
	"context"

	"github.com/gin-gonic/gin"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	pkgmiddleware "github.com/morehao/ark-iam/pkg/middleware"
	"github.com/morehao/ark-iam/rpapi/config"
	"github.com/morehao/ark-iam/rpapi/internal/router"
	"github.com/morehao/golib/biz/gserver/gindocs"
	"github.com/morehao/golib/glog"
)

const AppName = "rpapi"

// Init 装配 rpapi 应用。
//
// injectedKeySource 为 OP 公钥来源：gateway 单体部署由 auth 注入进程内 key set
// （零网络调用）；独立部署传 nil，此时按本应用配置自建（见 pkgmiddleware.ResolveKeySource）。
func Init(engine *gin.Engine, Conf *pkgconfig.Config, injectedKeySource pkgmiddleware.KeySource) {
	config.Conf = Conf
	keySource, keyErr := pkgmiddleware.ResolveKeySource(injectedKeySource, Conf)
	if keyErr != nil {
		// fail-closed：没有可用 KeySource 时目录路由照常挂载，但 OIDCAuth 会对每个
		// 请求返回 401（x-api-key 通道除外，它是独立的凭证校验）。
		// 刻意**不 panic**：gateway 聚合部署下 rpapi 与登录同进程，一个应用的密钥源
		// 配置问题不应把整个 IAM 打挂（受众与爆炸半径隔离的初衷）。
		glog.Errorf(context.Background(), "[%s.Init] oidc key source unavailable, err:%v", AppName, keyErr)
	}
	router.RegisterRouter(engine, Conf, keySource)
	if Conf != nil && Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}
}
