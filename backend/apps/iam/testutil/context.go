package testutil

import (
	"context"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/golib/biz/gcontext"
)

type contextKey string

func withCtxValue(ctx context.Context, key string, value any) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

func BuildIamContext(userID uint) context.Context {
	ctx := context.Background()
	user, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		panic(err)
	}
	if user == nil || user.ID == 0 {
		panic("user not found")
	}

	ctx = withCtxValue(ctx, gcontext.KeyUserID, user.ID)
	ctx = withCtxValue(ctx, gcontext.KeyTenantID, user.TenantID)
	ctx = withCtxValue(ctx, gcontext.KeyPersonID, user.PersonID)

	if user.TenantID > 0 {
		relation, err := dao.NewUserDepartmentDao().GetByCond(ctx, &dao.UserDepartmentCond{
			UserID: userID,
		})
		if err != nil {
			panic(err)
		}
		if relation != nil {
			ctx = withCtxValue(ctx, gcontext.KeyDeptID, relation.DepartmentID)
		}
	}

	return ctx
}
