package identity

import (
	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// OperatorFromGin 把当前请求的身份与元数据打包成业务输入 objidentity.Operator。
//
// 这是**唯一**的身份搬运点：controller 在绑定请求后调用一次，
// 把结果填进 DTO 的 Operator 字段，再把 c.Request.Context() 交给 service。
// service / dao 不 import 本包，也不读 gin 上下文。
//
// 非 HTTP 入口（后台任务、协议层回调）没有 gin ctx，应自行构造 Operator 或传零值。
//
// 取不到的值一律为空串，本函数不做 fail-closed——是否需要身份由各 service 按业务判定
// （需要身份的写操作应拒空，纯读操作可忽略）。
func OperatorFromGin(c *gin.Context) objidentity.Operator {
	if c == nil {
		return objidentity.Operator{}
	}
	return objidentity.Operator{
		UserID:    gincontext.GetUserIDString(c),
		PersonID:  gincontext.GetPersonIDString(c),
		ClientIP:  gincontext.GetClientIP(c),
		UserAgent: userAgentOf(c),
	}
}

// userAgentOf 读取请求的 UA。golib 的 gincontext 没有 GetUserAgent，
// 直接读 *http.Request；请求对象缺失时返回空串。
func userAgentOf(c *gin.Context) string {
	if c.Request == nil {
		return ""
	}
	return c.Request.UserAgent()
}
