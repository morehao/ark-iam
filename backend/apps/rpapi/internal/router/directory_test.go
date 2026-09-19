package router

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/rpapi/internal/dto/dtordirectory"
	"github.com/morehao/ark-iam/rpapi/testutil"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer   = "http://localhost:8081/oidc"
	testAudience = "rp-client"
	testTenantID = "tenant-1"
	testKID      = "rpapi-test-kid"
)

// newTestEngine 构造只挂 rpapi 目录路由的引擎，KeySource 用进程内 key set，
// 从而走与生产完全一致的验签链（本地验签），但不依赖 JWKS 网络。
func newTestEngine(t *testing.T, keySource rp.KeySource) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerRouter(engine, &config.Config{OIDC: config.OIDC{Issuer: testIssuer, Audiences: []string{testAudience}}}, keySource)
	return engine
}

// TestDirectoryFailsClosedWithoutKeySource 断言密钥源缺失（keySource == nil）时
// 目录路由仍然挂载但**不裸奔**：任何令牌都 401，且不 panic——
// gateway 聚合部署下 rpapi 与登录同进程，密钥源配置问题不得打挂整个 IAM。
func TestDirectoryFailsClosedWithoutKeySource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerRouter(engine, &config.Config{OIDC: config.OIDC{Issuer: testIssuer}}, nil)

	for _, token := range []string{"", "not-a-jwt"} {
		w := get(t, engine, "/v1/rp/directory/members", token, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.NotContains(t, w.Body.String(), "张三")
	}
}

func newKey(t *testing.T) (*rsa.PrivateKey, rp.KeySource) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key, rp.NewKeysFromSet(map[string]*rsa.PublicKey{testKID: &key.PublicKey})
}

func signM2MToken(t *testing.T, key *rsa.PrivateKey, scopes, tenantID, audience string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":         testAudience,
		"aud":         audience,
		"iss":         testIssuer,
		"exp":         time.Now().Add(time.Hour).Unix(),
		"iat":         time.Now().Unix(),
		"client_id":   audience,
		"token_usage": "machine",
		"scope":       scopes,
	}
	if tenantID != "" {
		claims["tenant_id"] = tenantID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKID
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func get(t *testing.T, engine *gin.Engine, path, token string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// directoryFixture 播种一个租户的成员/部门/角色，并返回可用于断言的 id。
type directoryFixture struct {
	memberID  string
	machineID string
	foreignID string
}

func seedDirectory(t *testing.T) directoryFixture {
	t.Helper()
	db := testutil.SetupSQLite(t,
		&model.UserEntity{},
		&model.DepartmentEntity{},
		&model.DepartmentUserEntity{},
		&model.RoleEntity{},
	)
	users := []model.UserEntity{
		{PersonID: "p-1", TenantID: testTenantID, UserType: model.UserTypeMember, Name: "张三", Status: model.UserStatusActive},
		{TenantID: testTenantID, UserType: model.UserTypeMachine, Name: "服务账号", Status: model.UserStatusSuspended},
		{TenantID: "tenant-2", UserType: model.UserTypeMember, Name: "别租户成员", Status: model.UserStatusActive},
	}
	require.NoError(t, db.Create(&users).Error)

	depts := []model.DepartmentEntity{
		{TenantID: testTenantID, Name: "总部", Status: model.DeptNodeStatusEnable, DeptDepth: 1},
		{TenantID: testTenantID, ParentID: users[0].ID, Name: "平台组", Status: model.DeptNodeStatusEnable, DeptDepth: 2},
	}
	require.NoError(t, db.Create(&depts).Error)
	relations := []model.DepartmentUserEntity{
		{TenantID: testTenantID, DepartmentID: depts[0].ID, UserID: users[0].ID, RelationType: model.DeptUserRelationPrimary},
		{TenantID: testTenantID, DepartmentID: depts[1].ID, UserID: users[0].ID, RelationType: model.DeptUserRelationSecondary},
	}
	require.NoError(t, db.Create(&relations).Error)
	require.NoError(t, db.Create(&model.RoleEntity{
		TenantID: testTenantID, AppID: "app-1", Name: "管理员", Code: "admin",
	}).Error)

	return directoryFixture{memberID: users[0].ID, machineID: users[1].ID, foreignID: users[2].ID}
}

// TestDirectoryRequiresScopeAndTenant 断言 401/403 的三个分支：
// 无令牌 → 401；缺 directory.read → 403；有 scope 但无 tenant_id → 403（绝不能冒成 500）。
func TestDirectoryRequiresScopeAndTenant(t *testing.T) {
	fx := seedDirectory(t)
	key, keySource := newKey(t)
	engine := newTestEngine(t, keySource)

	w := get(t, engine, "/v1/rp/directory/members", "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), fx.memberID)

	// 有合法令牌但缺 directory.read：403，且不泄露任何目录数据
	noScope := signM2MToken(t, key, "openid", testTenantID, testAudience)
	w = get(t, engine, "/v1/rp/directory/members", noScope, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotContains(t, w.Body.String(), fx.memberID)

	// 有 scope 但无 tenant_id：403
	noTenant := signM2MToken(t, key, "directory.read", "", testAudience)
	w = get(t, engine, "/v1/rp/directory/members", noTenant, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotContains(t, w.Body.String(), fx.memberID)

	// aud 不匹配：401（OIDCAuth 的 aud 收紧）
	wrongAud := signM2MToken(t, key, "directory.read", testTenantID, "another-client")
	w = get(t, engine, "/v1/rp/directory/members", wrongAud, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestDirectoryMembersTenantScoped 断言租户作用域：
// 本租户成员可见；跨租户 id 进 missing 且与不存在 id 不可区分；挂起成员照常返回。
func TestDirectoryMembersTenantScoped(t *testing.T) {
	fx := seedDirectory(t)
	key, keySource := newKey(t)
	engine := newTestEngine(t, keySource)
	token := signM2MToken(t, key, "directory.read", testTenantID, testAudience)

	w := get(t, engine, "/v1/rp/directory/members/"+fx.memberID, token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	// 目录 API 直接回裸 DTO（不是本仓业务接口的 {code,msg,data} 信封）：
	// 这样 ETag 哈希才与调用方实际拿到的字节一一对应。
	var single dtordirectory.Member
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &single))
	assert.Equal(t, fx.memberID, single.UserID)
	assert.Equal(t, "张三", single.Name)
	assert.Equal(t, "active", single.Status)
	assert.Equal(t, "member", single.UserType)
	assert.Equal(t, []string{"平台组", "总部"}, single.DepartmentNames)

	// 跨租户 id：404（与不存在不可区分）
	w = get(t, engine, "/v1/rp/directory/members/"+fx.foreignID, token, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, w.Body.String(), "别租户成员")

	// 批量：本租户两个 id 命中，跨租户与不存在都进 missing
	path := "/v1/rp/directory/members?ids=" + fx.memberID + "," + fx.machineID + "," + fx.foreignID + ",does-not-exist"
	w = get(t, engine, path, token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var batch dtordirectory.MemberListResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &batch))
	require.Len(t, batch.List, 2)
	assert.ElementsMatch(t, []string{fx.foreignID, "does-not-exist"}, batch.Missing,
		"跨租户 id 与不存在 id 必须不可区分")
	for _, item := range batch.List {
		if item.UserID == fx.machineID {
			// 挂起成员照常返回（目录只服务展示，绝不用 status 做放行判定）。
			assert.Equal(t, "suspended", item.Status)
			assert.Equal(t, "machine", item.UserType)
		}
	}
}

// TestDirectoryBatchOverLimitIs400 断言上限显式拒绝（不静默截断）。
func TestDirectoryBatchOverLimitIs400(t *testing.T) {
	seedDirectory(t)
	key, keySource := newKey(t)
	engine := newTestEngine(t, keySource)
	token := signM2MToken(t, key, "directory.read", testTenantID, testAudience)

	ids := make([]string, 0, 101)
	for i := 0; i < 101; i++ {
		ids = append(ids, "id-"+strconv.Itoa(i))
	}
	w := get(t, engine, "/v1/rp/directory/members?ids="+strings.Join(ids, ","), token, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDirectoryETagAndTreeAndRoles 覆盖 ETag/304 与另两个端点。
func TestDirectoryETagAndTreeAndRoles(t *testing.T) {
	seedDirectory(t)
	key, keySource := newKey(t)
	engine := newTestEngine(t, keySource)
	token := signM2MToken(t, key, "directory.read", testTenantID, testAudience)

	// 部门树
	w := get(t, engine, "/v1/rp/directory/departments/tree", token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "总部")
	etag := w.Header().Get("ETag")
	require.NotEmpty(t, etag, "必须返回 ETag")
	assert.Contains(t, w.Header().Get("Cache-Control"), "private")
	assert.Equal(t, "Authorization", w.Header().Get("Vary"))

	// If-None-Match 命中 → 304 且无 body
	w = get(t, engine, "/v1/rp/directory/departments/tree", token, map[string]string{"If-None-Match": etag})
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Empty(t, w.Body.String())

	// 角色清单：补齐 groups 只有编码的缺口
	w = get(t, engine, "/v1/rp/directory/roles", token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"admin"`)
	assert.Contains(t, w.Body.String(), `"name":"管理员"`)
	assert.Contains(t, w.Body.String(), `"truncated":false`)
}
