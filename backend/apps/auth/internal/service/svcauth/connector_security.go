package svcauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/morehao/ark-iam/pkg/iam/model"
)

// generatePKCEVerifier 生成 PKCE code_verifier（43 字符，CSPRNG，RFC 7636 §4.1）。
func generatePKCEVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// pkceChallengeS256 计算 PKCE S256 challenge。
func pkceChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// validateOutboundURL 校验连接器出站请求目标 URL：
//   - 仅允许 http/https；
//   - 解析后的 IP 不得为环回/私网/链路本地/未指定地址（SSRF 防护）。
//
// 注意：DNS 解析后到真正发起请求之间仍存在 TOCTOU（库内会再次解析），
// 严格的 DNS rebinding 防护需自定义 Transport 固定解析结果；此处作为基础防线。
func validateOutboundURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("unsupported url scheme: " + u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("empty url host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// 域名无法解析：请求层本身会因 DNS 失败而失败，不会触达内网，
		// 因此放行（真正要防的是"可解析到私网/环回 IP 的地址"）。
		return nil
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("forbidden target address: %s", ip)
		}
	}
	return nil
}

// validateConnectorRedirectURI 校验连接器授权回调地址：
//   - 必须与连接器配置的 redirectUri 同源（host 一致）；
//   - 必须为 https（回调地址出现在授权 URL 与 state 中，禁止 http 明文）。
//
// 防止客户端把授权码导向攻击者控制的回调地址（开放重定向/授权码劫持）。
func validateConnectorRedirectURI(redirectURI string, connectorEntity *model.ConnectorEntity) error {
	if redirectURI == "" {
		return errors.New("redirect uri is empty")
	}
	reqURL, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("parse redirect uri fail: %w", err)
	}
	if reqURL.Scheme != "https" {
		return errors.New("redirect uri must be https")
	}
	if reqURL.Hostname() == "" {
		return errors.New("redirect uri host is empty")
	}
	// 与配置的回调地址同源校验
	cfgRedirect := connectorConfiguredRedirectURI(connectorEntity)
	if cfgRedirect == "" {
		return errors.New("connector redirect uri not configured")
	}
	cfgURL, err := url.Parse(cfgRedirect)
	if err != nil {
		return fmt.Errorf("parse configured redirect uri fail: %w", err)
	}
	if !strings.EqualFold(reqURL.Hostname(), cfgURL.Hostname()) {
		return errors.New("redirect uri host not allowed")
	}
	return nil
}

// connectorConfiguredRedirectURI 从连接器 Config 中读取 redirectUri。
func connectorConfiguredRedirectURI(connectorEntity *model.ConnectorEntity) string {
	config, err := buildConnectorConfig(connectorEntity)
	if err != nil {
		return ""
	}
	return config.RedirectURI
}

// connectorSensitiveConfigKeys 连接器配置中不得返回给前端的敏感字段。
var connectorSensitiveConfigKeys = []string{
	"clientSecret",
	"client_secret",
	"accessToken",
	"refreshToken",
}

// sanitizeConnectorConfig 对返回给前端的连接器配置脱敏：删除敏感字段
// （clientSecret / token 等），防止凭据经 Detail / PageList 接口泄露。
// 返回脱敏后的配置对象（map），原配置不变。
func sanitizeConnectorConfig(config any) any {
	if config == nil {
		return map[string]any{}
	}
	m, ok := config.(map[string]any)
	if !ok {
		// 非对象配置（如数组/字符串）直接按原样返回
		return config
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isConnectorSensitiveKey(k) {
			out[k] = "******"
			continue
		}
		out[k] = v
	}
	return out
}

func isConnectorSensitiveKey(key string) bool {
	for _, k := range connectorSensitiveConfigKeys {
		if strings.EqualFold(key, k) {
			return true
		}
	}
	return false
}
