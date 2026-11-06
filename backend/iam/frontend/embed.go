// Package frontend 嵌入内嵌 IAM 的登录页静态产物（login-web dist），供 MountFrontend 托管。
package frontend

import "embed"

// Login 是登录/初始化页（login-web 构建产物）。其前端在根路径下访问 /v1、/oidc，
// 因此同源托管在与 API 相同的域名根即可直接工作（无需重建 base）。
//
//go:embed login
var Login embed.FS
