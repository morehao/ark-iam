package svcaudit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoaudit"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	auditTenantA = "tenant-a"
	auditTenantB = "tenant-b"
)

func newAuditFixture(t *testing.T) (*gorm.DB, AuditLogSvc) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.SetupSQLite(t, &model.AuditLogEntity{})
	return db, NewAuditLogSvc()
}

// seedAudit 播种审计行。测试库挂着与生产同一份租户隔离插件，跨租户播种需显式声明作用域；
// 实体自带 tenant_id，且插件不覆盖 INSERT。
func seedAudit(t *testing.T, db *gorm.DB, rows ...model.AuditLogEntity) {
	t.Helper()
	xdb := db.WithContext(dbclient.CrossTenantContext(context.Background()))
	for i := range rows {
		require.NoError(t, xdb.Create(&rows[i]).Error)
	}
}

func auditRow(id, tenantID, actorPersonID string, action model.AuditAction, result model.AuditResult, at time.Time) model.AuditLogEntity {
	return model.AuditLogEntity{
		BaseEntity:    gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}, CreatedAt: at, UpdatedAt: at},
		ActorPersonID: actorPersonID,
		ActorUserID:   "user-" + actorPersonID,
		TenantID:      tenantID,
		Action:        action,
		TargetType:    model.AuditTargetTypeUser,
		Result:        result,
		IP:            "10.0.0.1",
		Detail:        "detail-" + id,
	}
}

// TestPageListPlatformViewSeesAllTenants 是"跨租户作用域"的正向证明：
// ctx 声明的是「租户 A」作用域，平台视角仍必须看得到租户 B 的审计行
// （若误用默认注入的当前租户，这里只会返回 1 行）。同时验证时间倒序。
func TestPageListPlatformViewSeesAllTenants(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	seedAudit(t, db,
		auditRow("a1", auditTenantA, "person-a", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-time.Hour)),
		auditRow("b1", auditTenantB, "person-b", model.AuditActionLogout, model.AuditResultSuccess, now.Add(-2*time.Hour)),
	)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{})
	require.NoError(t, err)

	require.Len(t, resp.List, 2, "平台视角必须跨租户可见")
	assert.EqualValues(t, 2, resp.Total)
	assert.Equal(t, "a1", resp.List[0].AuditLogID, "必须按 created_at 倒序")
	assert.Equal(t, "b1", resp.List[1].AuditLogID)
	// 出参时间必须是秒级 unix 时间戳且非零。
	assert.Greater(t, resp.List[0].CreatedAt, int64(0))
	assert.InDelta(t, now.Add(-time.Hour).Unix(), resp.List[0].CreatedAt, 1)
}

// TestPageListTenantFilterHidesOtherTenants 按租户过滤时别家不可见（跨租户隔离的负向证明）。
func TestPageListTenantFilterHidesOtherTenants(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	seedAudit(t, db,
		auditRow("a1", auditTenantA, "person-a", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-time.Hour)),
		auditRow("a2", auditTenantA, "person-a", model.AuditActionLogout, model.AuditResultSuccess, now.Add(-2*time.Hour)),
		auditRow("b1", auditTenantB, "person-b", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-3*time.Hour)),
	)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{TenantID: auditTenantB})
	require.NoError(t, err)

	require.Len(t, resp.List, 1)
	assert.Equal(t, "b1", resp.List[0].AuditLogID)
	assert.Equal(t, auditTenantB, resp.List[0].TenantID)
	assert.EqualValues(t, 1, resp.Total, "总数也必须只算过滤后的租户")
	for _, item := range resp.List {
		assert.NotEqual(t, auditTenantA, item.TenantID, "租户 A 的行不得泄漏")
	}
}

// TestPageListActorAndTimeRangeFilter 按操作人 + 时间区间过滤（组合条件）。
func TestPageListActorAndTimeRangeFilter(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	seedAudit(t, db,
		auditRow("p1", auditTenantA, "person-x", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-3*time.Hour)),
		auditRow("p2", auditTenantA, "person-x", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-30*time.Hour)),
		auditRow("p3", auditTenantA, "person-y", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-2*time.Hour)),
	)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{
		ActorPersonID: "person-x",
		StartTime:     now.Add(-4 * time.Hour).Unix(),
		EndTime:       now.Add(-time.Hour).Unix(),
	})
	require.NoError(t, err)

	require.Len(t, resp.List, 1, "只有 person-x 且落在时间窗内的那一条")
	assert.Equal(t, "p1", resp.List[0].AuditLogID)
}

// TestPageListActionAndResultFilter 按动作 + 结果过滤。
func TestPageListActionAndResultFilter(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	seedAudit(t, db,
		auditRow("k1", auditTenantA, "person-a", model.AuditActionApiKeyCreate, model.AuditResultSuccess, now.Add(-time.Hour)),
		auditRow("k2", auditTenantA, "person-a", model.AuditActionApiKeyCreate, model.AuditResultFailure, now.Add(-2*time.Hour)),
		auditRow("k3", auditTenantA, "person-a", model.AuditActionApiKeyRevoke, model.AuditResultSuccess, now.Add(-3*time.Hour)),
	)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{
		Action: model.AuditActionApiKeyCreate,
		Result: model.AuditResultSuccess,
	})
	require.NoError(t, err)

	require.Len(t, resp.List, 1)
	assert.Equal(t, "k1", resp.List[0].AuditLogID)
}

// TestPageListPaginatesAndCapsPageSizePageSize 覆盖分页与单页上限：
// 传 pageSize=1000 也只返回 100 条（getPageListByCond 自身上限是 1000，这里的 100 是本接口的口径）。
func TestPageListPaginatesAndCapsPageSize(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	rows := make([]model.AuditLogEntity, 0, 120)
	for i := 0; i < 120; i++ {
		rows = append(rows, auditRow(fmt.Sprintf("row-%03d", i), auditTenantA, "person-a",
			model.AuditActionLogin, model.AuditResultSuccess, now.Add(-time.Duration(i)*time.Minute)))
	}
	seedAudit(t, db, rows...)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{
		PageQuery: gobject.PageQuery{Page: 1, PageSize: 1000},
	})
	require.NoError(t, err)
	assert.Len(t, resp.List, 100, "pageSize 上限 100")
	assert.EqualValues(t, 120, resp.Total, "total 是过滤后的全量条数，不受单页上限影响")

	// 第二页：起点正确（row-099 之后继续）。
	resp2, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{
		PageQuery: gobject.PageQuery{Page: 2, PageSize: 100},
	})
	require.NoError(t, err)
	assert.Len(t, resp2.List, 20)
	assert.Equal(t, "row-119", resp2.List[19].AuditLogID, "按时间倒序时最后一条是最旧的行")

	// 不传分页参数时用默认值，且绝不出现"不分页返回全部"（page/pageSize 非正数时 dao 会返回全部）。
	resp3, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{})
	require.NoError(t, err)
	assert.Len(t, resp3.List, auditLogDefaultPageSize)
}

// TestPageListDefaultsToRecentWindow 不传 startTime 时默认近 7 天：更早的数据需要显式传时间窗才可见。
func TestPageListDefaultsToRecentWindow(t *testing.T) {
	db, svc := newAuditFixture(t)
	now := time.Now()
	seedAudit(t, db,
		auditRow("recent", auditTenantA, "person-a", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-time.Hour)),
		auditRow("old", auditTenantA, "person-a", model.AuditActionLogin, model.AuditResultSuccess, now.Add(-30*24*time.Hour)),
	)

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	resp, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{})
	require.NoError(t, err)
	require.Len(t, resp.List, 1, "默认时间窗为近 7 天")
	assert.Equal(t, "recent", resp.List[0].AuditLogID)

	// 显式给出更早的窗口：历史数据可查。
	resp2, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{
		StartTime: now.Add(-60 * 24 * time.Hour).Unix(),
	})
	require.NoError(t, err)
	assert.Len(t, resp2.List, 2)
}

// TestPageListRejectsInvalidEnums 非法枚举入参必须报错，不能静默返回空集合。
func TestPageListRejectsInvalidEnums(t *testing.T) {
	db, svc := newAuditFixture(t)
	seedAudit(t, db, auditRow("a1", auditTenantA, "person-a", model.AuditActionLogin, model.AuditResultSuccess, time.Now()))

	ctx := testutil.NewGinCtx(auditTenantA, "operator")
	_, err := svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{Action: model.AuditAction("drop table")})
	assert.Equal(t, code.GetError(code.AuditLogPageListParamInvalidError), err)

	_, err = svc.PageList(ctx, &dtoaudit.AuditLogPageListReq{Result: model.AuditResult("maybe")})
	assert.Equal(t, code.GetError(code.AuditLogPageListParamInvalidError), err)
}
