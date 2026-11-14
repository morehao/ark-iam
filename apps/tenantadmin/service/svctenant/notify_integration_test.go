package svctenant

import (
	"context"
	"errors"
	"testing"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/testutil"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/notify"
	"github.com/morehao/golib/dbaccess/gormdao"
)

type fakeSenderTA struct {
	to  []string
	err error
}

func (f *fakeSenderTA) Send(_ context.Context, msg notify.Message) error {
	f.to = append(f.to, msg.To)
	return f.err
}

// 通道启用 + 投递成功：重置密码明文不回显、delivered=true。
func TestUserResetPasswordDeliversAndHidesPlaintext(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1", DeptPath: "/o1", DeptDepth: 1, Name: "部门o1", Status: model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	ginCtx := newAdminCtx(t, db, "t1", "op1")

	sender := &fakeSenderTA{}
	svc := &userSvc{db: dbclient.IamDB, notifier: notify.NewForTest(sender, nil, true)}

	created, err := svc.Create(requestCtx(ginCtx), &dtotenant.UserCreateReq{
		Name: "张三", PrimaryEmail: "zs@x.com", PrimaryDepartmentID: "o1", Operator: operatorOf(ginCtx),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	reset, err := svc.ResetPassword(requestCtx(ginCtx), &dtotenant.UserResetPasswordReq{
		UserID: created.UserID, Operator: operatorOf(ginCtx),
	})
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if reset.InitialPassword != "" {
		t.Errorf("initialPassword = %q, want empty when delivered", reset.InitialPassword)
	}
	if !reset.Delivered {
		t.Error("delivered = false, want true")
	}
	if len(sender.to) == 0 || sender.to[len(sender.to)-1] != "zs@x.com" {
		t.Errorf("delivery recipients = %v, want last=zs@x.com", sender.to)
	}
}

// 通道启用 + 投递失败 + requireDelivery=true：返回 NotifyDeliverError。
func TestUserResetPasswordDeliveryFailClosed(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{})
	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1", DeptPath: "/o1", DeptDepth: 1, Name: "部门o1", Status: model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	ginCtx := newAdminCtx(t, db, "t1", "op1")

	sender := &fakeSenderTA{} // 先让建用户成功投递
	svc := &userSvc{db: dbclient.IamDB, notifier: notify.NewForTest(sender, nil, true)}

	created, err := svc.Create(requestCtx(ginCtx), &dtotenant.UserCreateReq{
		Name: "张三", PrimaryEmail: "zs@x.com", PrimaryDepartmentID: "o1", Operator: operatorOf(ginCtx),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sender.err = errors.Join(notify.ErrDeliveryFailed, errors.New("sms down")) // 重置时投递失败
	_, err = svc.ResetPassword(requestCtx(ginCtx), &dtotenant.UserResetPasswordReq{
		UserID: created.UserID, Operator: operatorOf(ginCtx),
	})
	if !errors.Is(err, code.GetError(code.NotifyDeliverError)) {
		t.Fatalf("want NotifyDeliverError, got %v", err)
	}
}
