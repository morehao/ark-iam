package dtoapikey

import "github.com/morehao/golib/biz/gobject"

type CreateApiKeyReq struct {
	Name      string `json:"name" binding:"required"`
	Scope     string `json:"scope"`
	ExpiredAt string `json:"expiresAt"`
}

type RevokeApiKeyReq struct {
	ID uint `json:"id" binding:"required"`
}

type DeleteApiKeyReq struct {
	ID uint `json:"id" binding:"required"`
}

type ApiKeyPageListReq struct {
	gobject.PageQuery
	Name string `json:"name"`
}
