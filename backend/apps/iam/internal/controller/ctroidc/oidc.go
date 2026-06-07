package ctroidc

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcoidc"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type OIDCCtr struct {
	oidcAuthSvc svcoidc.OIDCAuthSvc
}

func NewOIDCCtr(provider *svcoidc.OIDCProvider) *OIDCCtr {
	return &OIDCCtr{oidcAuthSvc: svcoidc.NewOIDCAuthSvc(provider)}
}

func (ctr *OIDCCtr) Login(ctx *gin.Context) {
	var req dtooidc.OIDCLoginReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.oidcAuthSvc.CompleteLogin(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	if res.SessionID != "" {
		ttl := 86400
		if config.Conf != nil {
			ttl = config.Conf.OIDC.SessionTTL
		}
		if ttl <= 0 {
			ttl = 86400
		}
		ctx.SetCookie("iam_sso_session", res.SessionID, ttl, "/", "", false, true)
	}

	gincontext.Success(ctx, res)
}

func (ctr *OIDCCtr) SSOLogin(ctx *gin.Context) {
	authRequestID := ctx.Query("authRequestID")
	if authRequestID == "" {
		ctx.Redirect(302, config.Conf.OIDC.FrontendLoginURL)
		return
	}

	sessionID, err := ctx.Cookie("iam_sso_session")
	if err != nil {
		frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
		ctx.Redirect(302, frontendURL)
		return
	}

	continueURL, err := ctr.oidcAuthSvc.CompleteLoginBySession(ctx.Request.Context(), authRequestID, sessionID)
	if err != nil {
		ctx.SetCookie("iam_sso_session", "", -1, "/", "", false, true)
		frontendURL := config.Conf.OIDC.FrontendLoginURL + "?authRequestID=" + url.QueryEscape(authRequestID)
		ctx.Redirect(302, frontendURL)
		return
	}

	ctx.Redirect(302, continueURL)
}
