package oidcop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/glog"
)

// randomTokenID 生成 <prefix>-<32位十六进制随机串> 的标识（16 字节 CSPRNG）。
// 用于 auth request / access token / refresh token 等不可预测标识的生成，
// 替代此前基于 UnixNano 的可预测 ID。
func randomTokenID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b), nil
}

// accessTokenMetaKeyPrefix 是 access token 元数据在 Redis 的 key 前缀。
// 元数据在签发时写入，TTL 与 token 生命周期一致，用于：
//   - introspection 端点返回 scope/client_id/sub/exp 等完整信息（M1）；
//   - userinfo 端点按 token 实际 scope 裁剪声明（M2）。
const accessTokenMetaKeyPrefix = "iam:oidc:at:meta:"

func accessTokenMetaKey(tokenID string) string { return accessTokenMetaKeyPrefix + tokenID }

// accessTokenMeta 是一次 access token 签发时的上下文快照。
type accessTokenMeta struct {
	Subject    string    `json:"subject"`
	ClientID   string    `json:"clientID"`
	Scopes     []string  `json:"scopes"`
	IssuedAt   time.Time `json:"issuedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	TenantID   string    `json:"tenantID,omitempty"`
	SessionID  string    `json:"sessionID,omitempty"`
	TokenUsage string    `json:"tokenUsage,omitempty"`
	Username   string    `json:"username,omitempty"`
}

// storeAccessTokenMeta 尽力写入 access token 元数据；Redis 不可用时仅记日志，
// 不阻断签发（introspection/userinfo 降级为不返回 scope 相关声明）。
func storeAccessTokenMeta(ctx context.Context, tokenID string, meta accessTokenMeta) {
	if dbclient.RedisCli == nil || tokenID == "" {
		return
	}
	data, err := json.Marshal(meta)
	if err != nil {
		glog.Warnf(ctx, "[oidcop.storeAccessTokenMeta] marshal fail, tokenID:%s, err:%v", tokenID, err)
		return
	}
	if err := dbclient.RedisCli.Set(ctx, accessTokenMetaKey(tokenID), data, metaTTLFor(meta.ExpiresAt)).Err(); err != nil {
		glog.Warnf(ctx, "[oidcop.storeAccessTokenMeta] redis set fail, tokenID:%s, err:%v", tokenID, err)
	}
}

// loadAccessTokenMeta 读取 access token 元数据；不存在（过期/未知 token）返回 nil。
func loadAccessTokenMeta(ctx context.Context, tokenID string) *accessTokenMeta {
	if dbclient.RedisCli == nil || tokenID == "" {
		return nil
	}
	data, err := dbclient.RedisCli.Get(ctx, accessTokenMetaKey(tokenID)).Bytes()
	if err != nil {
		return nil
	}
	var meta accessTokenMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		glog.Warnf(ctx, "[oidcop.loadAccessTokenMeta] unmarshal fail, tokenID:%s, err:%v", tokenID, err)
		return nil
	}
	return &meta
}

// metaTTLFor 计算元数据 TTL：token 剩余有效期，另加时钟余量。
func metaTTLFor(expiration time.Time) time.Duration {
	ttl := time.Until(expiration) + 5*time.Minute
	if ttl <= 0 {
		return time.Minute
	}
	return ttl
}
