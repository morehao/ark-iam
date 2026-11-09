package credential

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"unicode"
	"unicode/utf8"
)

// 本文件承载「低熵口令」一侧：人工输入、熵低，必须用 bcrypt 慢哈希存储
// （哈希由调用方走 gcrypto.GeneratePasswordHash）。高熵机密的生成与摘要见 secret.go。

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

// 临时密码字符集与长度：剔除易混淆字符（数字 0/1、字母 l/I/O），
// 降低人工转录（运营复制转交）时的错录概率。
const (
	temporaryLength = 16
	lowerChars      = "abcdefghijkmnopqrstuvwxyz"
	upperChars      = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars      = "23456789"
)

// GenerateTemporaryPassword 生成一次性临时密码（每用户随机），满足 ValidateStrength。
//
// 规则（全系统唯一实现，任何临时/默认口令都必须来自这里）：
//   - 长度固定 16 字符，字符集 = 小写字母 + 大写字母 + 数字（已剔除易混淆字符）；
//   - 前 3 位强制分别取一个小写、一个大写、一个数字，保证必然通过强度校验；
//   - 其余位从全字符集随机取，最后对整体做 Fisher-Yates 洗牌，避免"必含字符"位置可预测；
//   - 随机源为 crypto/rand（CSPRNG），非 math/rand。
//
// 调用方职责：bcrypt 哈希后落库；明文只允许放入创建/重置响应（或交由消息通道下发），
// 禁止写入 glog / 审计日志 / 其他任何落盘；也不得回写为"可再查询"的明文。
//
// TODO(delivery): 接入邮件/短信通道后，临时密码改为由通道下发给账号本人，
// 创建/重置响应不再回显明文。这是本文件所有"一次性回显"约定的收敛点：
// 全仓 grep TODO(delivery) 可定位到每个待改造的回显位置。
// 见 docs/design/system-design.md §5.8。
func GenerateTemporaryPassword() (string, error) {
	allChars := lowerChars + upperChars + digitChars

	buf := make([]byte, 0, temporaryLength)
	for _, charSet := range []string{lowerChars, upperChars, digitChars} {
		char, err := randChar(charSet)
		if err != nil {
			return "", err
		}
		buf = append(buf, char)
	}
	for len(buf) < temporaryLength {
		char, err := randChar(allChars)
		if err != nil {
			return "", err
		}
		buf = append(buf, char)
	}
	if err := shuffle(buf); err != nil {
		return "", err
	}

	generated := string(buf)
	// 防御性自检：生成规则若被改坏（如字符集写错），在此立即暴露，而不是把弱口令发出去。
	if err := ValidateStrength(generated); err != nil {
		return "", fmt.Errorf("generated temporary password is weak: %w", err)
	}
	return generated, nil
}

// randChar 从字符集中等概率取一个字符（crypto/rand）。
func randChar(charSet string) (byte, error) {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
	if err != nil {
		return 0, fmt.Errorf("read crypto/rand fail: %w", err)
	}
	return charSet[idx.Int64()], nil
}

// shuffle 原地 Fisher-Yates 洗牌（crypto/rand）。
func shuffle(buf []byte) error {
	for i := len(buf) - 1; i > 0; i-- {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("read crypto/rand fail: %w", err)
		}
		j := idx.Int64()
		buf[i], buf[j] = buf[j], buf[i]
	}
	return nil
}
