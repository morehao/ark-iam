package connector

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
)

// IsValidStatus 校验连接器状态取值：空值表示「不指定/不修改」，非空必须命中白名单常量。
// 校验归 service（AGENTS.md 硬规则 3）：DTO 绑定前端原始字符串，非法值在此拦截。
func IsValidStatus(status model.ConnectorStatus) bool {
	switch status {
	case "", model.ConnectorStatusEnable, model.ConnectorStatusDisable:
		return true
	default:
		return false
	}
}

// normalizeSwitch 校验并归一化单个连接器开关枚举：空串按 disable 处理
// （列默认值即 disable，保持「未提交即关闭」的既有行为），非空必须命中白名单常量。
func normalizeSwitch[T ~string](value T, enable T, disable T) (T, bool) {
	switch value {
	case "":
		return disable, true
	case enable, disable:
		return value, true
	default:
		return "", false
	}
}

// NormalizeSwitches 就地校验并归一化连接器的 4 个开关列
// （allow_auto_create_user / allow_account_link / sync_profile / enable_token_storage），
// 任一非法值即整体返回 false，由调用方返回该操作既有的功能级错误码。
func NormalizeSwitches(info *objauth.ConnectorBaseInfo) bool {
	var ok bool
	if info.AllowAutoCreateUser, ok = normalizeSwitch(info.AllowAutoCreateUser, model.ConnectorAutoCreateUserFlagEnable, model.ConnectorAutoCreateUserFlagDisable); !ok {
		return false
	}
	if info.AllowAccountLink, ok = normalizeSwitch(info.AllowAccountLink, model.ConnectorAccountLinkFlagEnable, model.ConnectorAccountLinkFlagDisable); !ok {
		return false
	}
	if info.SyncProfile, ok = normalizeSwitch(info.SyncProfile, model.ConnectorSyncProfileFlagEnable, model.ConnectorSyncProfileFlagDisable); !ok {
		return false
	}
	if info.EnableTokenStorage, ok = normalizeSwitch(info.EnableTokenStorage, model.ConnectorTokenStorageFlagEnable, model.ConnectorTokenStorageFlagDisable); !ok {
		return false
	}
	return true
}
