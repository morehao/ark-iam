package testutil

import (
	"context"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/golib/biz/gcontext"
)

func BuildIamContext(userID uint) context.Context {
	ctx := context.Background()
	user, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		panic(err)
	}
	if user == nil || user.ID == 0 {
		panic("user not found")
	}

	ctx = context.WithValue(ctx, gcontext.KeyUserID, user.ID)
	ctx = context.WithValue(ctx, gcontext.KeyTenantID, user.TenantID)
	ctx = context.WithValue(ctx, gcontext.KeyPersonID, user.PersonID)

	if user.TenantID > 0 {
		relation, err := dao.NewUserDepartmentRelationDao().GetByCond(ctx, &dao.UserDepartmentRelationCond{
			UserID:    userID,
		})
		if err != nil {
			panic(err)
		}
		if relation != nil {
			ctx = context.WithValue(ctx, gcontext.KeyDeptID, relation.DepartmentID)
		}
	}

	return ctx
}

func ptrInt8(v int8) *int8 {
	return &v
}