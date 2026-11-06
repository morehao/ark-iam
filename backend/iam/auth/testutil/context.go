package testutil

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/testkit"
)

func WithIamContext(userID string) testkit.Option {
	return func(gc *gin.Context) {
		// 测试助手按自然人反查其归属租户，此刻租户未知：显式声明「全部租户」作用域。
		user, err := dao.NewUserDao().GetByID(dbclient.CrossTenantContext(context.Background()), userID)
		if err != nil {
			panic(err)
		}
		if user == nil || user.ID == "" {
			panic("user not found")
		}

		gc.Set(gcontext.KeyUserID, user.ID)
		// 租户作用域：类型化值 + gin Keys 投影，与生产中间件写法完全一致。
		gincontext.SetTenantScope(gc, gcontext.CurrentScope(user.TenantID))
		gc.Set(gcontext.KeyPersonID, user.PersonID)
	}
}
