// Package svcdb 承载 auth 域各 service 包共享的默认 DBGetter。
// 可嵌入 IAM 库下由 iam.App 在 Mount 前注入（一进程一实例语义）；
// 供包级测试 seam（如 authLoginRecorder / insertOIDCLoginLog）使用。
package svcdb

import "github.com/morehao/golib/dbaccess/gormdao"

// DB 是 auth 域的默认数据访问器，由 iam.App 设置。
var DB gormdao.DBGetter

// Set 由 iam.App 注入。
func Set(db gormdao.DBGetter) { DB = db }
