// Package apikey 提供 API Key 的生成与哈希工具，供平台/租户端密钥管理共享复用。
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Generate 生成一段新的明文 API Key（32 字节随机数的十六进制，共 64 位）。
func Generate() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Hash 计算明文密钥的 SHA-256 哈希（数据库仅存哈希）。
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Prefix 返回明文密钥前缀（前 7 位），用于展示与识别。
func Prefix(raw string) string {
	if len(raw) < 7 {
		return raw
	}
	return raw[:7]
}
