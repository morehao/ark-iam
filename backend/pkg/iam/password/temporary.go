package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// BootstrapAdminPassword 种子平台管理员（bootstrap）的固定初始口令。
//
// 这是全系统**唯一**允许存在的固定默认口令，且只在种子阶段使用：
//   - 平台租户由 pkg/seed 在启动时创建，其管理员的初始口令必须可被部署文档与
//     e2e 用例稳定引用（e2e/config.ts 即以该口令登录），因此刻意保留固定值；
//   - 其余一切"代建 / 重置"场景的初始口令都必须走 GenerateTemporary（每用户随机）。
//
// 不要把本常量用于任何业务链路；新增业务默认口令一律用 GenerateTemporary。
const BootstrapAdminPassword = "admin123"

// 临时密码字符集与长度：剔除易混淆字符（数字 0/1、字母 l/I/O），
// 降低人工转录（运营复制转交）时的错录概率。
const (
	temporaryLength = 16
	lowerChars      = "abcdefghijkmnopqrstuvwxyz"
	upperChars      = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars      = "23456789"
)

// GenerateTemporary 生成一次性临时密码（每用户随机），满足 ValidateStrength。
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
// 见 docs/design/tenant-admin-provisioning-design-20260912.md T1（通道立项）、Q4（通道排期）。
func GenerateTemporary() (string, error) {
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
