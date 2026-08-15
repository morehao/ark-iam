package testutil

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/testkit"
)

func WithIamContext(userID string) testkit.Option {
	return func(gc *gin.Context) {
		user, err := dao.NewUserDao().GetByID(context.Background(), userID)
		if err != nil {
			panic(err)
		}
		if user == nil || user.ID == "" {
			panic("user not found")
		}

		gc.Set(gcontext.KeyUserID, user.ID)
		gc.Set(gcontext.KeyTenantID, user.TenantID)
		gc.Set(gcontext.KeyPersonID, user.PersonID)
	}
}
