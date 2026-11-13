package dtoaudit

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gobject"
)

// AuditLogPageListReq 审计日志分页查询入参（只读；平台视角，默认跨租户）。
//
// 过滤维度刻意收敛：租户 / 操作人 / 动作 / 结果 / 时间区间。
// 目标 id 与关键词检索不在本轮（方案 §13：审计检索走运维侧，控制台只做定位式查询）。
type AuditLogPageListReq struct {
	gobject.PageQuery
	// TenantID 租户ID（空=不限租户，平台视角可见全部租户的审计）。
	TenantID string `json:"tenantID" form:"tenantID"`
	// ActorPersonID 操作人自然人ID（空=不限操作人）。
	ActorPersonID string `json:"actorPersonID" form:"actorPersonID"`
	// Action 动作标识（空=不限）；取值白名单见 model.AuditAction*。
	Action model.AuditAction `json:"action" form:"action"`
	// Result 结果（空=不限）；取值白名单见 model.AuditResult*。
	Result model.AuditResult `json:"result" form:"result"`
	// StartTime 起始时间（Unix 秒，含）；不传时默认为近 7 天，防止无界扫描全表。
	StartTime int64 `json:"startTime" form:"startTime"`
	// EndTime 结束时间（Unix 秒，含）；不传表示不设上界。
	EndTime int64 `json:"endTime" form:"endTime"`
}
