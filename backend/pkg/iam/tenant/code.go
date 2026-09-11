package tenant

import (
	"fmt"

	"github.com/morehao/golib/gutil"
)

// 租户编码规则：t_<12 位随机小写 hex>，例：t_3f7a9c1d2e4b。
// 平台建租户（platformadmin）与自助开通租户（auth 注册）共用 GenerateCode，
// 保证两条创建路径产出的编码形态一致；编码一经生成不再变更（Update 不修改 code）。
const (
	// CodePrefix 租户编码固定前缀（tenant 首字母），便于人工识别与检索。
	CodePrefix = "t"
	// codeRandomBytes 随机段字节数：6 字节 → 12 位 hex（48bit 随机，
	// 万级租户规模下碰撞概率 < 1e-6，且 tenant.code 唯一索引兜底）。
	codeRandomBytes = 6
)

// GenerateCode 生成租户编码：t_<12 位随机 hex>。
// 随机段由 golib gutil.RandomHex（crypto/rand）产生，不含时间信息。
func GenerateCode() (string, error) {
	randomPart, err := gutil.RandomHex(codeRandomBytes)
	if err != nil {
		return "", fmt.Errorf("generate tenant code random part: %w", err)
	}
	return fmt.Sprintf("%s_%s", CodePrefix, randomPart), nil
}
