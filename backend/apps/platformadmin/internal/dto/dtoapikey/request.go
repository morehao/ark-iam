package dtoapikey

import "github.com/morehao/golib/biz/gobject"

type ApiKeyCreateReq struct {
	Name      string `json:"name" binding:"required"`
	Scope     string `json:"scope"`
	ExpiredAt string `json:"expiresAt"`
}

type RevokeApiKeyReq struct {
	ApiKeyID uint `json:"-" uri:"apiKeyID" binding:"required"`
}

type ApiKeyDeleteReq struct {
	ApiKeyID uint `json:"-" uri:"apiKeyID" binding:"required"`
}

type ApiKeyPageListReq struct {
	gobject.PageQuery
	Name string `json:"name" form:"name"`
}
