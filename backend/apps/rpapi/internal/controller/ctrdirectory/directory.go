// Package ctrdirectory 是只读目录 API 的控制器层。
//
// 缓存契约（设计文档关键决策六）：
//   - ETag 是**序列化后响应体的哈希**（不用 max(updated_at)：软删不改 updated_at
//     会让已删成员在客户端长期滞留，staleness 变成无界）；
//   - If-None-Match 命中 → 304（省带宽与序列化，**不省 DB 查询**）；
//   - Cache-Control: private, no-cache + Vary: Authorization（显式声明不进共享缓存）。
package ctrdirectory

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/rpapi/internal/dto/dtordirectory"
	"github.com/morehao/ark-iam/rpapi/internal/service/svcdirectory"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// DirectoryCtr 是只读目录控制器接口。
type DirectoryCtr interface {
	Member(ctx *gin.Context)
	Members(ctx *gin.Context)
	DepartmentTree(ctx *gin.Context)
	Roles(ctx *gin.Context)
}

type directoryCtr struct {
	directorySvc svcdirectory.DirectorySvc
}

var _ DirectoryCtr = (*directoryCtr)(nil)

// NewDirectoryCtr 构造只读目录控制器。
func NewDirectoryCtr() DirectoryCtr {
	return &directoryCtr{directorySvc: svcdirectory.NewDirectorySvc()}
}

// @Tags 目录（RP）
// @Summary 成员摘要
// @Produce application/json
// @Param userID path string true "成员用户ID"
// @Success 200 {object} gincontext.DtoRender{data=dtordirectory.Member}
// @Router /v1/rp/directory/members/{userID} [get]
func (ctr *directoryCtr) Member(ctx *gin.Context) {
	userID := ctx.Param("userID")
	member, err := ctr.directorySvc.Member(ctx, userID)
	if err != nil {
		glog.Errorf(ctx, "[ctrdirectory.Member] svc fail, err:%v, userID:%s", err, userID)
		fail(ctx, code.RpDirectoryMembersError)
		return
	}
	if member == nil {
		// 跨租户与不存在不可区分（防枚举）。
		fail(ctx, code.RpDirectoryNotFoundError)
		return
	}
	writeJSON(ctx, member)
}

// @Tags 目录（RP）
// @Summary 批量成员摘要
// @Produce application/json
// @Param ids query string true "逗号分隔的成员用户ID，最多 100 个"
// @Success 200 {object} gincontext.DtoRender{data=dtordirectory.MemberListResp}
// @Router /v1/rp/directory/members [get]
func (ctr *directoryCtr) Members(ctx *gin.Context) {
	ids := parseIDs(ctx.Query("ids"))
	if len(ids) > dtordirectory.MaxBatchMemberIDs {
		// 显式拒绝，不静默截断（否则调用方会以为"没返回就是不存在"）。
		fail(ctx, code.RpDirectoryBadRequestError)
		return
	}
	resp, err := ctr.directorySvc.Members(ctx, ids)
	if err != nil {
		glog.Errorf(ctx, "[ctrdirectory.Members] svc fail, err:%v, count:%d", err, len(ids))
		fail(ctx, code.RpDirectoryMembersError)
		return
	}
	writeJSON(ctx, resp)
}

// @Tags 目录（RP）
// @Summary 部门树
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=[]dtordirectory.Department}
// @Router /v1/rp/directory/departments/tree [get]
func (ctr *directoryCtr) DepartmentTree(ctx *gin.Context) {
	tree, err := ctr.directorySvc.DepartmentTree(ctx)
	if err != nil {
		glog.Errorf(ctx, "[ctrdirectory.DepartmentTree] svc fail, err:%v", err)
		fail(ctx, code.RpDirectoryDepartmentError)
		return
	}
	writeJSON(ctx, tree)
}

// @Tags 目录（RP）
// @Summary 角色清单
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtordirectory.RoleListResp}
// @Router /v1/rp/directory/roles [get]
func (ctr *directoryCtr) Roles(ctx *gin.Context) {
	resp, err := ctr.directorySvc.Roles(ctx)
	if err != nil {
		glog.Errorf(ctx, "[ctrdirectory.Roles] svc fail, err:%v", err)
		fail(ctx, code.RpDirectoryRolesError)
		return
	}
	writeJSON(ctx, resp)
}

// writeJSON 以稳定 JSON 序列化 + ETag 协商写响应。
//
// 用 gutil.ToJsonString（而非 ctx.JSON）是为了让"算哈希的字节"与"发出去的字节"
// 严格同一份——否则 ETag 与实际内容可能不一致，304 就会把错误内容永久固化。
func writeJSON(ctx *gin.Context, payload any) {
	body := gutil.ToJsonString(payload)
	sum := sha256.Sum256([]byte(body))
	etag := `"` + base64.RawURLEncoding.EncodeToString(sum[:16]) + `"`

	ctx.Header("ETag", etag)
	// 带 Authorization 的响应显式声明不进共享缓存（RFC 9111 §3.5）。
	ctx.Header("Cache-Control", "private, no-cache")
	ctx.Header("Vary", "Authorization")

	if match := ctx.GetHeader("If-None-Match"); match != "" && etagMatches(match, etag) {
		ctx.Status(http.StatusNotModified)
		return
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(body))
}

// etagMatches 支持逗号分隔的候选列表与 W/ 弱前缀（保守比较：去弱前缀后精确匹配）。
func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		if strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

// parseIDs 解析逗号分隔的 id 列表（去空白、去空项、去重、保序）。
func parseIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

// fail 按错误码映射到设计文档固定的 HTTP 状态码语义（401/403/404/400/503），
// 其余系统错误统一 503（SDK 侧对应 ErrDirectoryUnavailable，可 stale 降级）。
func fail(ctx *gin.Context, errorCode int) {
	status := statusForErrorCode(errorCode)
	ctx.AbortWithStatusJSON(status, gin.H{
		"code":      errorCode,
		"requestID": gincontext.GetRequestID(ctx),
		"msg":       messageForErrorCode(errorCode),
		"data":      nil,
	})
}

func statusForErrorCode(errorCode int) int {
	switch errorCode {
	case code.RpDirectoryUnauthorizedError:
		return http.StatusUnauthorized
	case code.RpDirectoryForbiddenError:
		return http.StatusForbidden
	case code.RpDirectoryNotFoundError:
		return http.StatusNotFound
	case code.RpDirectoryBadRequestError:
		return http.StatusBadRequest
	default:
		return http.StatusServiceUnavailable
	}
}

// messageForErrorCode 复用 code 包注册的消息，缺失时给一个非空兜底。
func messageForErrorCode(errorCode int) string {
	entry := code.GetError(errorCode)
	if entry.Msg != "" {
		return entry.Msg
	}
	return "目录服务暂不可用 (" + strconv.Itoa(errorCode) + ")"
}
