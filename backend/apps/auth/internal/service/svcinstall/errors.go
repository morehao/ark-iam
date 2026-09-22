package svcinstall

import (
	"errors"
	"net/http"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/gerror"
)

// 初始化入口的业务错误哨兵。
//
// 全部使用 install 领域错误码（107xxx），并由 HTTPStatusOf 映射到**真实 HTTP 状态码**：
// 调用方是浏览器页面与自动化部署脚本，它们按状态码决定"该做什么"，而不是解析业务码。
//
// 注意其中两条刻意复用 install 段码值、但带更具体的文案（联系方式缺失）：
// 页面需要直接告诉运维"邮箱与手机号至少填一个"，而单一 InstallBadRequest 文案说不出这点。
// 复用码值不损失可编程性——状态码与码值仍是 400 + InstallationBadRequestError。
var (
	// errAlreadyInitialized 系统已初始化，禁止重复初始化（409）。
	errAlreadyInitialized = code.GetError(code.InstallationAlreadyInitializedError)
	// errBadRequest 入参非法（用户名缺失或超长等，400）。
	errBadRequest = code.GetError(code.InstallationBadRequestError)
	// errContactRequired 管理员邮箱与手机号至少一项（400）。
	errContactRequired = gerror.Error{
		Code: code.InstallationBadRequestError,
		Msg:  "管理员邮箱与手机号至少填写一项",
	}
	// errPasswordWeak 口令不满足强度要求（400）。
	errPasswordWeak = code.GetError(code.InstallationPasswordWeakError)
	// errTokenNotConfigured 服务端未配置 BOOTSTRAP_TOKEN，端点不可用（503）。
	errTokenNotConfigured = code.GetError(code.InstallationTokenNotConfiguredError)
	// errTokenInvalid 令牌缺失或不匹配（401）。
	errTokenInvalid = code.GetError(code.InstallationTokenInvalidError)
)

// HTTPStatusOf 把业务错误映射为 HTTP 状态码与业务码。
//
// 未登记为 install 领域错误的按 500 处理（"服务端出错"是唯一安全的默认：
// 把它当 400 会把系统错误说成用户输错了）。
func HTTPStatusOf(err error) (int, int) {
	var ge gerror.Error
	if errors.As(err, &ge) {
		if status := code.InstallErrorHTTPStatus(ge.Code); status != 0 {
			return status, ge.Code
		}
	}
	return http.StatusInternalServerError, code.InstallationInitializeError
}

// BusinessCodeOf 提取 install 领域业务码；非本领域错误返回 0。
//
// 与 HTTPStatusOf 分开是因为两者用途不同：前者决定"怎么回给客户端"（必须有状态码），
// 后者用于审计明细与日志（未登记错误要能被识别为"不是业务错误"而不是伪装成 107003）。
func BusinessCodeOf(err error) int {
	var ge gerror.Error
	if errors.As(err, &ge) && code.InstallErrorHTTPStatus(ge.Code) != 0 {
		return ge.Code
	}
	return 0
}

// CheckBootstrapToken 校验请求头携带的引导令牌与部署配置是否一致。
//
// 返回 nil 表示通过。三类失败各自的语义（也决定了页面该提示什么）：
//   - 服务端未配置 BOOTSTRAP_TOKEN：**整体不可用**（503）。刻意 fail-closed——
//     若改成"未配置则跳过校验"，一个忘记配置 token 的部署就等于把
//     "任何能访问 /install 的人都可以创建平台管理员"变成真实风险；
//   - 请求未携带令牌：401；
//   - 令牌不匹配：401（与"未携带"同码，避免向未认证方泄露"你差一点就对了"）。
//
// 口令与令牌都属敏感值，本函数与调用方**都不得把令牌写入日志**。
//
// 两侧都做首尾空白归一：部署时 token 常由 `export T=$(cat file)`、docker secret 或页面粘贴
// 带进来尾随换行/空格，若因此判"不匹配"，运维会看到一个无法自查的 401
// （令牌肉眼完全一致）。代价是"以空白开头/结尾的令牌"无法使用——可接受，
// 且对称归一后不存在"只归一一边"导致的隐性不等。
func CheckBootstrapToken(provided string) error {
	expected := trimToken(envBootstrapToken())
	if expected == "" {
		return errTokenNotConfigured
	}
	if subtleCompare(trimToken(provided), expected) {
		return nil
	}
	return errTokenInvalid
}
