package dtoauth

import "github.com/morehao/ark-iam/iam/object/objauth"

type LoginResp struct {
	objauth.TokenInfo
}

type RegisterResp struct {
	UserID uint `json:"userID"` // 用户ID
}

type RefreshTokenResp struct {
	objauth.TokenInfo
}

type UserinfoResp struct {
	objauth.UserInfo
}

type SsoAuthorizationUrlResp struct {
	AuthorizationUrl string `json:"authorizationUrl"` // 授权地址
}

type SsoCallbackResp struct {
	objauth.TokenInfo
}