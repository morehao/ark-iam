package code

import "github.com/morehao/golib/gerror"

// install 领域错误码段（初始化引导入口，/install/*）。
//
// 与其它领域不同，本段必须使用**真实 HTTP 状态码**（经 gincontext.FailWithStatus 返回），
// 因为调用方是浏览器页面与自动化部署脚本：部署脚本按 409/401/503 判断"该做什么"，
// 若与业务错误一样恒返回 200，脚本就不得不解析业务码，而且任何中间件/网关都无法按
// 状态码做处置（重试、告警、拒绝）。状态映射收口在 InstallErrorHTTPStatus。
//
// 码值语义与设计文档一致（107xxx 段，rp 段止于 106007）：
//
//	409 已初始化（终态，不可绕过）
//	401 令牌缺失或不匹配
//	503 服务端未配置 BOOTSTRAP_TOKEN（端点整体不可用，fail-closed）
//	500 初始化执行失败（系统错误，已记日志）
//	409 守卫拦截：系统尚未初始化（业务端点在此之前不可用）
//	400 入参非法 / 口令不满足强度
const (
	// InstallationAlreadyInitializedError 系统已初始化，禁止重复初始化（HTTP 409）。
	// 这是**终态**：初始化成功后该端点永久返回此错误，不因进程重启而解除。
	InstallationAlreadyInitializedError = 107000
	// InstallationTokenInvalidError bootstrap token 缺失或不匹配（HTTP 401）。
	InstallationTokenInvalidError = 107001
	// InstallationTokenNotConfiguredError 服务端未配置 BOOTSTRAP_TOKEN，端点不可用（HTTP 503）。
	// fail-closed：未配置时不是"跳过校验"，而是整体拒绝——否则一个忘记配置 token 的部署
	// 会把"任何能访问 /install 的人都可以创建平台管理员"变成真实风险。
	InstallationTokenNotConfiguredError = 107002
	// InstallationInitializeError 初始化执行失败（系统错误，已记日志；HTTP 500）。
	InstallationInitializeError = 107003
	// InstallationPendingError 守卫拦截：系统尚未初始化，业务端点不可用（HTTP 409）。
	// 前端据此把用户引导到 /install。
	InstallationPendingError = 107004
	// InstallationBadRequestError 入参非法（缺联系方式、username 不合法等；HTTP 400）。
	InstallationBadRequestError = 107005
	// InstallationPasswordWeakError 管理员口令不满足 credential.ValidateStrength（HTTP 400）。
	InstallationPasswordWeakError = 107006
)

// installErrorHTTPStatus 错误码 → HTTP 状态码。未登记的业务码回落 400（入参类），
// 系统级错误由调用方显式指定 500，不在此表。
var installErrorHTTPStatus = map[int]int{
	InstallationAlreadyInitializedError: 409,
	InstallationTokenInvalidError:       401,
	InstallationTokenNotConfiguredError: 503,
	InstallationInitializeError:         500,
	InstallationPendingError:            409,
	InstallationBadRequestError:         400,
	InstallationPasswordWeakError:       400,
}

// InstallErrorHTTPStatus 返回 /install 领域错误码对应的 HTTP 状态码。
//
// 未登记的码返回 0，调用方据此判定"这不是 install 领域错误"并走默认处置——
// 返回 0 而不是 400 是为了不把未知错误伪装成"入参非法"（会把根因掩盖成用户错误）。
func InstallErrorHTTPStatus(code int) int {
	return installErrorHTTPStatus[code]
}

// installErrorMsgMap 错误码文案（与上面常量一一对应）。
var installErrorMsgMap = gerror.CodeMsgMap{
	InstallationAlreadyInitializedError: "系统已完成初始化，无法重复初始化",
	InstallationTokenInvalidError:       "初始化令牌缺失或不正确",
	InstallationTokenNotConfiguredError: "服务端未配置 BOOTSTRAP_TOKEN，初始化接口不可用",
	InstallationInitializeError:         "初始化执行失败，请检查服务端日志",
	InstallationPendingError:            "系统尚未初始化，请先完成初始化",
	InstallationBadRequestError:         "初始化参数不合法",
	InstallationPasswordWeakError:       "管理员口令强度不足（须包含大写字母、小写字母与数字）",
}
