package code

import (
	"testing"
)

// TestInstallErrorHTTPStatus 初始化接口的 HTTP 状态码是**对外契约**：
// 浏览器页面与自动化部署脚本按它决定"该做什么"（409 已初始化 / 401 换 token /
// 503 提示配置 env / 400 改输入），因此每一个码的状态都必须钉住。
func TestInstallErrorHTTPStatus(t *testing.T) {
	want := map[int]int{
		InstallationAlreadyInitializedError: 409,
		InstallationTokenInvalidError:       401,
		InstallationTokenNotConfiguredError: 503,
		InstallationInitializeError:         500,
		InstallationPendingError:            409,
		InstallationBadRequestError:         400,
		InstallationPasswordWeakError:       400,
	}
	codes := []int{
		InstallationAlreadyInitializedError,
		InstallationTokenInvalidError,
		InstallationTokenNotConfiguredError,
		InstallationInitializeError,
		InstallationPendingError,
		InstallationBadRequestError,
		InstallationPasswordWeakError,
	}
	for _, c := range codes {
		if got := InstallErrorHTTPStatus(c); got != want[c] {
			t.Errorf("InstallErrorHTTPStatus(%d) = %d, want %d", c, got, want[c])
		}
		if _, ok := errorMap[c]; !ok {
			t.Errorf("错误码 %d 未注册（gerror 取不到文案）", c)
		}
		if msg := errorMap[c].Msg; msg == "" {
			t.Errorf("错误码 %d 文案为空", c)
		}
	}
	// 未登记的业务码必须返回 0（调用方据此走默认处置），不得伪装成 400 入参错误——
	// 那会把"未知错误"掩盖成"用户输入有问题"。
	if got := InstallErrorHTTPStatus(999999); got != 0 {
		t.Errorf("未登记错误码的状态 = %d, want 0", got)
	}
}

// TestInstallErrorCodesAreContiguousAndInRange 码值落在 1070XX 段内且连续，
// 便于人工核对"有没有漏登记/跨段占用"（rp 段止于 106007）。
func TestInstallErrorCodesAreContiguousAndInRange(t *testing.T) {
	for i, c := range []int{
		InstallationAlreadyInitializedError,
		InstallationTokenInvalidError,
		InstallationTokenNotConfiguredError,
		InstallationInitializeError,
		InstallationPendingError,
		InstallationBadRequestError,
		InstallationPasswordWeakError,
	} {
		if c != 107000+i {
			t.Errorf("install 错误码不连续: 第 %d 个 = %d, want %d", i, c, 107000+i)
		}
	}
}
