package dtooauthclient

import "github.com/morehao/ark-iam/iam/object/objoauthclient"

type CreateResp struct {
	OAuthClientID uint   `json:"oauthClientId"`
	ClientID      string `json:"clientID"`
}

type DetailResp struct {
	OAuthClientID uint `json:"oauthClientId"`
	objoauthclient.OAuthClientBaseInfo
}

type PageListResp struct {
	List  []PageListItem `json:"list"`
	Total int64          `json:"total"`
}

type PageListItem struct {
	OAuthClientID uint `json:"oauthClientId"`
	objoauthclient.OAuthClientBaseInfo
}

type SecretResp struct {
	ID            uint64  `json:"id"`
	OAuthClientID uint64  `json:"oauthClientId"`
	Name          string  `json:"name"`
	ValuePrefix   string  `json:"valuePrefix"`
	ExpiredAt     *string `json:"expiresAt"`
	CreatedAt     string  `json:"createdAt"`
}

type SecretListResp struct {
	Total   int64        `json:"total"`
	Secrets []SecretResp `json:"secrets"`
}

type CreateSecretResp struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	ValuePrefix string `json:"valuePrefix"`
	Secret      string `json:"secret"`
}
