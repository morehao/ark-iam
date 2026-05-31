package ctroidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
)

type OIDCCtr struct {
	provider *svcoidc.OIDCProvider
}

func NewOIDCCtr(provider *svcoidc.OIDCProvider) *OIDCCtr {
	return &OIDCCtr{provider: provider}
}

func (ctr *OIDCCtr) LoginCallback(ctx *gin.Context) {
	authReqID := ctx.PostForm("authRequestID")
	if authReqID == "" {
		ctx.String(http.StatusBadRequest, "missing auth request id")
		return
	}
	ctx.Redirect(http.StatusFound, "/v1/iam/oidc/authorize/callback?id="+authReqID)
}
