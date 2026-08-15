// Package gctx 提供从 gin 上下文读取字符串类型身份标识的便捷函数。
//
// 自 string-id 改造起，personID/userID/tenantID 均为字符串主键（UUID v7），
// 与 golib gincontext 包中返回 uint 的 GetPersonID/GetUserID/GetTenantID 对应。
package gctx

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/gcontext"
)

// GetPersonID 从 gin 上下文读取自然人 ID（字符串主键）。
func GetPersonID(ctx *gin.Context) string {
	return ctx.GetString(gcontext.KeyPersonID)
}

// GetUserID 从 gin 上下文读取用户 ID（字符串主键）。
func GetUserID(ctx *gin.Context) string {
	return ctx.GetString(gcontext.KeyUserID)
}

// GetTenantID 从 gin 上下文读取租户 ID（字符串主键）。
func GetTenantID(ctx *gin.Context) string {
	return ctx.GetString(gcontext.KeyTenantID)
}
