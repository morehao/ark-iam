package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	oidcPromptNone         = "none"
	oidcErrorLoginRequired = "login_required"
)

func SilentSSORequired(ssoCookieName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Query("prompt") != oidcPromptNone {
			ctx.Next()
			return
		}
		if _, err := ctx.Cookie(ssoCookieName); err == nil {
			ctx.Next()
			return
		}

		redirectURI := ctx.Query("redirect_uri")
		if redirectURI == "" {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		u, err := url.Parse(redirectURI)
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		state := ctx.Query("state")
		q := u.Query()
		q.Set("error", oidcErrorLoginRequired)
		if state != "" {
			q.Set("state", state)
		}
		u.RawQuery = q.Encode()
		ctx.Redirect(http.StatusFound, u.String())
		ctx.Abort()
	}
}
