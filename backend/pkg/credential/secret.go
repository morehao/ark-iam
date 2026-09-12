// Package credential 提供系统内凭证的生成、校验与摘要。
//
// 按「熵」分两类，哈希策略不同、不可混用：
//   - 高熵机密（secret.go）：机器生成的随机串，如 API Key、OAuth client secret、
//     refresh token。熵足够（≥128 bit），用 SHA-256 快哈希存储即可，明文一律不落库；
//   - 低熵口令（password.go）：人工输入的口令，必须用 bcrypt 慢哈希抗离线暴力破解，
//     哈希本身由调用方走 golib 的 gcrypto.GeneratePasswordHash。
//
// 两侧的判据在 secret.go / password.go 各自文件头再次说明，避免"为什么 API Key
// 不用 bcrypt"变成需要考古的问题。
package credential

import (
	"encoding/hex"

	"github.com/morehao/golib/gcrypto"
)

// 各凭证类型的形态常量：随机字节数与展示前缀长度属于"凭证形态策略"，
// 而非落库字典值，因此集中在本包维护，避免调用方各自写魔数。
const (
	// APIKeyBytes API Key 的随机字节数（32 字节 = 256 bit 熵）。
	APIKeyBytes = 32
	// APIKeyPrefixLen API Key 展示前缀长度（前 7 位），用于列表识别。
	APIKeyPrefixLen = 7

	// ClientSecretBytes OAuth client secret 的随机字节数（32 字节 = 256 bit 熵）。
	ClientSecretBytes = 32
	// ClientSecretPrefixLen OAuth client secret 展示前缀长度（前 8 位）。
	ClientSecretPrefixLen = 8
)

// GenerateSecret 生成高熵机密：byteLen 字节 CSPRNG（crypto/rand）随机数的小写 hex 串。
//
// 调用方职责与口令一致：明文只允许在"创建/重置"响应中回显一次（或交由消息通道下发），
// 禁止写入 glog / 审计日志 / 其他任何落盘；落库只允许 HashSecret 的结果。
func GenerateSecret(byteLen int) (string, error) {
	buf, err := gcrypto.GenerateRandomBytes(byteLen)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashSecret 计算高熵机密的 SHA-256 摘要（小写 hex，全长不截断）。
//
// 这是全系统唯一的机密摘要入口：落库与比对都必须经本函数。
// 算法或编码一旦变化，存量库中已签发的 API Key / OAuth client secret / refresh token
// 会立即全部失效且不可逆——改动前必须先过 credential 包的黄金向量测试。
func HashSecret(raw string) string {
	return gcrypto.SHA256Hash(raw)
}

// Prefix 返回明文的展示前缀（前 n 位），用于列表识别与人工比对。
// n 取各凭证类型的常量（APIKeyPrefixLen / ClientSecretPrefixLen）；
// n 非正或明文短于 n 时原样返回，保证短输入不 panic。
func Prefix(s string, n int) string {
	if n <= 0 || len(s) < n {
		return s
	}
	return s[:n]
}
