package svctenant

import (
	"testing"

	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// 通道未启用（gsetting 无 notify 配置 / enabled=false）：重置密码回显明文、delivered=false。
//
// 投递成功 / requireDelivery 收窄等「渠道可用」分支已由 pkg/notify 单测覆盖；
// service 层验证「未启用 → 回显」这一确定性映射。
func TestUserResetPasswordNotifyDisabledKeepsEcho(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{}, &model.PersonEntity{}, &model.DepartmentEntity{},
		&model.DepartmentUserEntity{}, &model.RoleEntity{}, &model.UserRoleEntity{}, &model.RefreshTokenEntity{})
	if err := db.Create(&model.DepartmentEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "o1"}},
		TenantID:   "t1", DeptPath: "/o1", DeptDepth: 1, Name: "部门o1", Status: model.DeptNodeStatusEnable,
	}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	ginCtx := newAdminCtx(t, db, "t1", "op1")

	svc := &userSvc{db: dbclient.IamDB} // 空 notify 配置 = 通道未启用

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
	if reset.InitialPassword == "" {
		t.Error("initialPassword empty, want echo when channel disabled")
	}
	if reset.Delivered {
		t.Error("delivered = true, want false when channel disabled")
	}
}
