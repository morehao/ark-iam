// Package password 提供密码强度校验的公共实现，
// 供注册（svcauth）与修改密码（svcperson）共用同一规则，避免规则漂移。
package password

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

const (
	// MinLength 密码最小长度（字符数）。
	MinLength = 8
	// MaxLength 密码最大长度（字符数）。bcrypt 有效输入上限为 72 字节，
	// 超长输入只会徒增计算成本，且可被用来放大登录端 DoS。
	MaxLength = 128
)

// ErrTooShort / ErrTooLong / ErrWeak 供调用方区分失败原因。
var (
	ErrTooShort = errors.New("password too short")
	ErrTooLong  = errors.New("password too long")
	ErrWeak     = errors.New("password must contain uppercase, lowercase and digit")
)

// ValidateStrength 校验密码强度：8~128 个字符，且同时包含大写、小写、数字。
// 使用 RuneCountInString 按字符数（而非字节数）统计长度。
func ValidateStrength(password string) error {
	if n := utf8.RuneCountInString(password); n < MinLength {
		return ErrTooShort
	} else if n > MaxLength {
		return ErrTooLong
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeak
	}
	return nil
}
