package svctenant

import (
	"regexp"
	"testing"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

// TestTenantPageListReturnsTimeFields 列表必须同时回传创建时间与更新时间
// （前端「创建时间」「更新时间」两列都读这两个字段，缺失则渲染为 "-"）。
func TestTenantPageListReturnsTimeFields(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	if err := db.Create(&model.TenantEntity{
		Code: "t_000000000001",
		Name: "Acme Corp",
		Type: model.TenantTypeCustomer,
	}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := &tenantSvc{}
	resp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("PageList len = %d, want 1", len(resp.List))
	}
	item := resp.List[0]
	if item.CreatedAt <= 0 {
		t.Errorf("createdAt = %d, want > 0", item.CreatedAt)
	}
	if item.UpdatedAt <= 0 {
		t.Errorf("updatedAt = %d, want > 0", item.UpdatedAt)
	}
	if item.UpdatedAt < item.CreatedAt {
		t.Errorf("updatedAt(%d) < createdAt(%d)", item.UpdatedAt, item.CreatedAt)
	}
}

// TestTenantPageListNameKeyword 列表按租户名模糊搜索：name 入参映射到条件 Keyword。
func TestTenantPageListNameKeyword(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	for _, entity := range []*model.TenantEntity{
		{Code: "t_000000000002", Name: "Acme Corp", Type: model.TenantTypeCustomer},
		{Code: "t_000000000003", Name: "Globex", Type: model.TenantTypeCustomer},
	} {
		if err := db.Create(entity).Error; err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	svc := &tenantSvc{}
	resp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Name: "cme"})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(resp.List) != 1 || resp.List[0].Name != "Acme Corp" {
		t.Fatalf("PageList keyword result = %+v, want only Acme Corp", resp.List)
	}
}

// TestTenantCreateGeneratesCode 建租户时编码由服务端按规则自动生成，客户端传值被忽略。
func TestTenantCreateGeneratesCode(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{}, &model.OrganizationEntity{})

	svc := &tenantSvc{}
	req := &dtotenant.TenantCreateReq{}
	req.Name = "Acme Corp"
	req.Type = string(model.TenantTypeCustomer)
	req.Code = "manual-code-should-be-ignored"

	resp, err := svc.Create(newTenantScopeGinCtx(""), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.TenantID == "" {
		t.Fatal("Create returned empty tenantID")
	}

	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), resp.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil {
		t.Fatal("created tenant not found")
	}
	pattern := regexp.MustCompile(`^t_[0-9a-f]{12}$`)
	if !pattern.MatchString(stored.Code) {
		t.Errorf("stored code = %q, want match %s", stored.Code, pattern.String())
	}
	if stored.Code == req.Code {
		t.Errorf("stored code = %q, want server-generated code ignoring req.Code", stored.Code)
	}
}

// TestTenantCreateNormalizesStatus 租户状态是白名单枚举：非法值/缺省一律归一为 active，
// 绝不把脏值落库（脏值会让 IsActive 判定为不可用，导致租户整体无法登录）。
func TestTenantCreateNormalizesStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{}, &model.OrganizationEntity{})

	svc := &tenantSvc{}
	req := &dtotenant.TenantCreateReq{}
	req.Name = "Acme Corp"
	req.Type = string(model.TenantTypeCustomer)
	req.Status = model.TenantStatus("bogus")

	resp, err := svc.Create(newTenantScopeGinCtx(""), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), resp.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil {
		t.Fatal("created tenant not found")
	}
	if stored.Status != model.TenantStatusActive {
		t.Errorf("stored status = %q, want %q", stored.Status, model.TenantStatusActive)
	}
}

// TestTenantUpdateSuspendsAndListsStatus 挂起他租户：状态落库并在列表出参回传
// （前端「状态」列读 status，不再读已废弃的 isSuspended）。
func TestTenantUpdateSuspendsAndListsStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{}, &model.OrganizationEntity{})

	svc := &tenantSvc{}
	createReq := &dtotenant.TenantCreateReq{}
	createReq.Name = "Acme Corp"
	createReq.Type = string(model.TenantTypeCustomer)
	created, err := svc.Create(newTenantScopeGinCtx(""), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updateReq := &dtotenant.TenantUpdateReq{TenantID: created.TenantID}
	updateReq.Name = "Acme Corp"
	updateReq.Type = string(model.TenantTypeCustomer)
	updateReq.Status = model.TenantStatusSuspended
	// 操作者所在租户是另一个租户，允许挂起
	if err := svc.Update(newTenantScopeGinCtx("other-tenant"), updateReq); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), created.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil || stored.Status != model.TenantStatusSuspended {
		t.Fatalf("stored status = %+v, want %q", stored, model.TenantStatusSuspended)
	}

	listResp, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	var found bool
	for _, item := range listResp.List {
		if item.TenantID == created.TenantID {
			found = true
			if item.Status != model.TenantStatusSuspended {
				t.Errorf("list status = %q, want %q", item.Status, model.TenantStatusSuspended)
			}
		}
	}
	if !found {
		t.Fatalf("suspended tenant not found in list: %+v", listResp.List)
	}
}

// TestTenantUpdateRefusesSuspendOwnTenant 禁止挂起操作者自己所在的租户：
// 挂起后该租户无法登录、平台控制台随之失联且无恢复路径（不可逆自锁）。
func TestTenantUpdateRefusesSuspendOwnTenant(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{}, &model.OrganizationEntity{})

	svc := &tenantSvc{}
	createReq := &dtotenant.TenantCreateReq{}
	createReq.Name = "Acme Corp"
	createReq.Type = string(model.TenantTypeCustomer)
	created, err := svc.Create(newTenantScopeGinCtx(""), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updateReq := &dtotenant.TenantUpdateReq{TenantID: created.TenantID}
	updateReq.Name = "Acme Corp"
	updateReq.Type = string(model.TenantTypeCustomer)
	updateReq.Status = model.TenantStatusSuspended
	err = svc.Update(newTenantScopeGinCtx(created.TenantID), updateReq)
	if err == nil || err.Error() != code.GetError(code.TenantSuspendSelfForbiddenError).Error() {
		t.Fatalf("Update err = %v, want %v", err, code.GetError(code.TenantSuspendSelfForbiddenError))
	}

	// 拒绝后状态不得被改写
	stored, err := dao.NewTenantDao().GetByID(newTenantScopeGinCtx(""), created.TenantID)
	if err != nil {
		t.Fatalf("dao GetByID failed: %v", err)
	}
	if stored == nil || stored.Status != model.TenantStatusActive {
		t.Fatalf("stored status = %+v, want %q", stored, model.TenantStatusActive)
	}
}

// TestTenantPageListStatusFilter 状态筛选：不传表示不筛选，传 active/suspended 精确过滤。
func TestTenantPageListStatusFilter(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.TenantEntity{})
	for _, entity := range []*model.TenantEntity{
		{Code: "t_000000000011", Name: "Active Co", Type: model.TenantTypeCustomer, Status: model.TenantStatusActive},
		{Code: "t_000000000012", Name: "Suspended Co", Type: model.TenantTypeCustomer, Status: model.TenantStatusSuspended},
	} {
		if err := db.Create(entity).Error; err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	svc := &tenantSvc{}
	all, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{})
	if err != nil {
		t.Fatalf("PageList(no filter) failed: %v", err)
	}
	if len(all.List) != 2 {
		t.Fatalf("PageList(no filter) len = %d, want 2", len(all.List))
	}

	suspended, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatusSuspended})
	if err != nil {
		t.Fatalf("PageList(suspended) failed: %v", err)
	}
	if len(suspended.List) != 1 || suspended.List[0].Name != "Suspended Co" {
		t.Fatalf("PageList(suspended) = %+v, want only Suspended Co", suspended.List)
	}
	if suspended.List[0].Status != model.TenantStatusSuspended {
		t.Errorf("filtered status = %q, want %q", suspended.List[0].Status, model.TenantStatusSuspended)
	}

	active, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatusActive})
	if err != nil {
		t.Fatalf("PageList(active) failed: %v", err)
	}
	if len(active.List) != 1 || active.List[0].Name != "Active Co" {
		t.Fatalf("PageList(active) = %+v, want only Active Co", active.List)
	}
}

// TestTenantPageListRejectsInvalidStatus 非法状态筛选值必须报错，而不是静默返回全部/空集
// （静默处理会让调用方以为"筛选成功但没数据"）。
func TestTenantPageListRejectsInvalidStatus(t *testing.T) {
	testutil.SetupSQLite(t, &model.TenantEntity{})

	svc := &tenantSvc{}
	_, err := svc.PageList(newTenantScopeGinCtx(""), &dtotenant.TenantPageListReq{Status: model.TenantStatus("bogus")})
	if err == nil || err.Error() != code.GetError(code.TenantPageListStatusInvalidError).Error() {
		t.Fatalf("PageList err = %v, want %v", err, code.GetError(code.TenantPageListStatusInvalidError))
	}
}
