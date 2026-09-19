package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth"
	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/platformadmin"
	"github.com/morehao/ark-iam/rpapi"
	"github.com/morehao/ark-iam/tenantadmin"
)

const AppName = "gateway"

func Init(engine *gin.Engine, Conf *config.Config) {
	// auth 先装配 OP，并返回本进程已发布公钥的进程内 KeySource（含过渡期旧 key）。
	// gateway 聚合部署下，同进程的其余应用直接复用它做本地验签：
	//   - 零网络调用：不去 HTTP 请求自己的 JWKS 端点（进程启动期服务尚未监听，
	//     "自取公钥"必然失败）；
	//   - 自动跟随多 key 轮换，无需挂载 OP 私钥、无需重启。
	keySource := auth.Init(engine, Conf)
	platformadmin.Init(engine, Conf, keySource)
	tenantadmin.Init(engine, Conf, keySource)
	rpapi.Init(engine, Conf, keySource)
}
