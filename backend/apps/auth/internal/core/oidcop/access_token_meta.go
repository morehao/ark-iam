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

// accessTokenRevokedKeyPrefix 是 access token 撤销黑名单的 key 前缀。
// JWT access token 无法在签发侧物理撤销，只能经黑名单使 OP 侧的
// userinfo / introspection 端点拒绝（RFC 7009 撤销语义）。
const accessTokenRevokedKeyPrefix = "iam:oidc:at:revoked:"

func accessTokenMetaKey(tokenID string) string    { return accessTokenMetaKeyPrefix + tokenID }
func accessTokenRevokedKey(tokenID string) string { return accessTokenRevokedKeyPrefix + tokenID }

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

// revokeAccessToken 将 access token 的 jti 写入撤销黑名单。
// TTL 尽量对齐 token 剩余有效期（读取元数据推算），元数据不可得时保守用 15 分钟。
// 黑名单仅作用于 OP 侧的 userinfo / introspection 端点（RP 侧无状态依赖短 TTL）。
func revokeAccessToken(ctx context.Context, tokenID string) {
	if dbclient.RedisCli == nil || tokenID == "" {
		return
	}
	ttl := 15 * time.Minute
	if meta := loadAccessTokenMeta(ctx, tokenID); meta != nil {
		if d := time.Until(meta.ExpiresAt); d > 0 {
			ttl = d + time.Minute
		}
	}
	if err := dbclient.RedisCli.Set(ctx, accessTokenRevokedKey(tokenID), "1", ttl).Err(); err != nil {
		glog.Warnf(ctx, "[oidcop.revokeAccessToken] redis set fail, tokenID:%s, err:%v", tokenID, err)
	}
}

// isAccessTokenRevoked 判断 access token 是否已被撤销（命中黑名单）。
// Redis 不可用/未知 token 时返回 false（fail-open：黑名单是尽力而为的撤销增强）。
func isAccessTokenRevoked(ctx context.Context, tokenID string) bool {
	if dbclient.RedisCli == nil || tokenID == "" {
		return false
	}
	n, err := dbclient.RedisCli.Exists(ctx, accessTokenRevokedKey(tokenID)).Result()
	return err == nil && n > 0
}
