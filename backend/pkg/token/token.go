package token

import (
	"crypto/sha256"
	"fmt"
)

// HashToken 计算 token 的 SHA-256 十六进制摘要，用于数据库存储与比对
// （明文 token 不落库，仅存哈希）。
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
