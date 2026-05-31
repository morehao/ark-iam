package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/controller/ctroidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gserver/ginserver"
)

func InitOIDC(engine *gin.Engine, groups *ginserver.RouterGroups) {
	issuer := fmt.Sprintf("http://localhost:%s/v1/iam/oidc", config.Conf.Server.Port)

	provider, err := svcoidc.SetupOIDCProvider(issuer)
	if err != nil {
		panic(fmt.Sprintf("failed to setup OIDC provider: %v", err))
	}

	ctr := ctroidc.NewOIDCCtr(provider)

	v1Group := groups.MustGetGroup(gconstant.ApiVersionV1)
	oidcGroup := v1Group.Group("/oidc")
	oidcGroup.Any("/*path", gin.WrapH(http.StripPrefix("/v1/iam/oidc", provider.GetProviderHTTPHandler())))

	v1Group.POST("/oidc/login/callback", ctr.LoginCallback)
}
