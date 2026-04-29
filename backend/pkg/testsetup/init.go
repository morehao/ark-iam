package testsetup

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/testkit"
)

type Initializer = testkit.Initializer
type InitializerFunc = testkit.InitializerFunc

var (
	mu                  sync.RWMutex
	initializerCreators = map[string]InitializerFunc{
		AppNameDemo: newDemoappInitializer,
		AppNameIam:  newIamappInitializer,
	}
	registeredApps = make(map[string]bool)
)

func init() {
	gin.SetMode(gin.TestMode)
}

func RegisterApp(appName string, initFunc InitializerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registeredApps[appName] {
		return
	}
	testkit.RegisterInitializer(appName, initFunc)
	registeredApps[appName] = true
}

func Initialize(appName string) {
	mu.Lock()
	if !registeredApps[appName] {
		if creator, ok := initializerCreators[appName]; ok {
			testkit.RegisterInitializer(appName, creator)
			registeredApps[appName] = true
		}
	}
	mu.Unlock()
	testkit.Initialize(appName)
}

func Close(appName string) {
	testkit.Close(appName)
}

func Init(appName string) {
	Initialize(appName)
}

func Done(appName string) {
	Close(appName)
}

func NewContext(opts ...testkit.Option) *gin.Context {
	return testkit.NewContext(opts...)
}

func WithAuthByUserID(userID uint) testkit.Option {
	return func(ctx *gin.Context) {
		db := dbclient.IamDB(ctx)

		type userInfo struct {
			ID       uint
			TenantID uint
			DeptID   uint
			PersonID uint
		}
		var user userInfo
		if err := db.Raw("SELECT id, tenant_id, dept_id, person_id FROM iam_user WHERE id = ? AND deleted_at IS NULL", userID).Scan(&user).Error; err != nil {
			panic(fmt.Sprintf("WithAuthByUserID: query user failed, userID=%d, err=%v", userID, err))
		}
		if user.ID == 0 {
			panic(fmt.Sprintf("WithAuthByUserID: user not found, userID=%d", userID))
		}

		ctx.Set(gcontext.KeyUserID, userID)
		ctx.Set(gcontext.KeyTenantID, user.TenantID)
		ctx.Set(gcontext.KeyDeptID, user.DeptID)
		ctx.Set(gcontext.KeyPersonID, user.PersonID)

		if user.TenantID > 0 {
			type tenantInfo struct {
				OrgID uint
			}
			var tenant tenantInfo
			if err := db.Raw("SELECT org_id FROM iam_tenant WHERE id = ? AND deleted_at IS NULL", user.TenantID).Scan(&tenant).Error; err != nil {
				panic(fmt.Sprintf("WithAuthByUserID: query tenant failed, tenantID=%d, err=%v", user.TenantID, err))
			}
			if tenant.OrgID > 0 {
				ctx.Set(gcontext.KeyOrgID, tenant.OrgID)
			}
		}
	}
}
