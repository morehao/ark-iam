package code

import "github.com/morehao/golib/gerror"

const (
	OIDCInvalidRequest          = 100790
	OIDCUnauthorizedClient      = 100791
	OIDCAccessDenied            = 100792
	OIDCUnsupportedResponseType = 100793
	OIDCInvalidScope            = 100794
	OIDCInvalidGrant            = 100795
	OIDCInvalidClient           = 100796
	OIDCServerError             = 100797
	OIDCTemporarilyUnavailable  = 100798
	OIDCSessionNotFound         = 100799
)

var oidcErrorMsgMap = gerror.CodeMsgMap{
	OIDCInvalidRequest:          "OIDC invalid request",
	OIDCUnauthorizedClient:      "OIDC unauthorized client",
	OIDCAccessDenied:            "OIDC access denied",
	OIDCUnsupportedResponseType: "OIDC unsupported response type",
	OIDCInvalidScope:            "OIDC invalid scope",
	OIDCInvalidGrant:            "OIDC invalid grant",
	OIDCInvalidClient:           "OIDC invalid client",
	OIDCServerError:             "OIDC server error",
	OIDCTemporarilyUnavailable:  "OIDC temporarily unavailable",
	OIDCSessionNotFound:         "OIDC session not found",
}
