package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newIndexProbeDB 建一个只含 audit_log 的内存库并跑 AutoMigrate + EnsurePerformanceIndexes，
// 与生产启动路径（AutoMigrateAll）保持一致的口径。
func newIndexProbeDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:audit_index_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, AutoMigrateAll(db))

	// 播种足够行数，避免计划器因"空表/行数极少"选择全表扫描而让本测试失去意义。
	rows := make([]AuditLogEntity, 0, 200)
	for i := 0; i < 200; i++ {
		at := time.Now().Add(-time.Duration(i) * time.Minute)
		rows = append(rows, AuditLogEntity{
			BaseEntity:    gormdao.BaseEntity{CreatedAt: at, UpdatedAt: at},
			TenantID:      fmt.Sprintf("tenant-%02d", i%4),
			ActorPersonID: fmt.Sprintf("person-%02d", i%10),
			Action:        AuditActionLogin,
			Result:        AuditResultSuccess,
		})
	}
	require.NoError(t, db.Create(&rows).Error)
	require.NoError(t, db.Exec("ANALYZE").Error)
	return db
}

// TestAuditLogPerformanceIndexesCreated 三条审计查询索引确实被建出（P0 T6.2）。
func TestAuditLogPerformanceIndexesCreated(t *testing.T) {
	db := newIndexProbeDB(t)
	for _, name := range []string{"idx_audit_tenant_created", "idx_audit_created", "idx_audit_actor_created"} {
		var count int64
		require.NoError(t, db.Raw(
			"select count(*) from sqlite_master where type='index' and tbl_name='audit_log' and name=?", name).Scan(&count).Error)
		assert.EqualValues(t, 1, count, "索引 %s 未创建", name)
	}
}

// TestAuditLogQueryPlanUsesIndex 是 T6 的 EXPLAIN 验收：
// 三种查询形态（租户视角/全平台视角/操作人视角）都必须走索引，而不是全表扫描（Seq Scan）。
//
// 说明：EXPLAIN 证据在 SQLite 上以 EXPLAIN QUERY PLAN 采集（可在 CI 复现）；
// PostgreSQL 侧由部署后 `EXPLAIN (ANALYZE, BUFFERS)` 复核，SQL 形态与这里逐字一致。
func TestAuditLogQueryPlanUsesIndex(t *testing.T) {
	db := newIndexProbeDB(t)

	cases := []struct {
		name      string
		query     string
		wantIndex string
	}{
		{
			name:      "租户视角（tenant_id + 时间窗，审计列表默认形态）",
			query:     "select * from audit_log where tenant_id='tenant-01' and created_at >= '2026-01-01' and created_at <= '2027-01-01' order by created_at desc, id desc limit 20",
			wantIndex: "idx_audit_tenant_created",
		},
		{
			name:      "全平台视角（仅时间窗）",
			query:     "select * from audit_log where created_at >= '2026-01-01' and created_at <= '2027-01-01' order by created_at desc, id desc limit 20",
			wantIndex: "idx_audit_created",
		},
		{
			name:      "操作人视角（actor_person_id + 时间窗）",
			query:     "select * from audit_log where actor_person_id='person-01' and created_at >= '2026-01-01' and created_at <= '2027-01-01' order by created_at desc, id desc limit 20",
			wantIndex: "idx_audit_actor_created",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var plans []struct {
				ID     int
				Parent int
				Detail string
			}
			require.NoError(t, db.Raw("EXPLAIN QUERY PLAN "+c.query).Scan(&plans).Error)
			require.NotEmpty(t, plans, "EXPLAIN QUERY PLAN 必须有输出")

			details := make([]string, 0, len(plans))
			for _, p := range plans {
				details = append(details, p.Detail)
			}
			plan := strings.Join(details, " | ")
			t.Logf("查询计划：%s", plan)

			assert.Contains(t, plan, c.wantIndex, "必须命中预期索引")
			assert.NotContains(t, plan, "SCAN audit_log", "不得退化为全表扫描")
		})
	}
}
