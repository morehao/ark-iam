package oidckit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/sdk/rp"
)

// goldenModulusHex 是用于 kid 派生的固定模数。deriveKeyID 只用到 N，
// 因此无需真实私钥即可锁定算法口径。
const goldenModulusHex = "c3530e0d1a5d0b5f2e7c4a9b8d6f3c2e1a4b7d9f0c2e5a8b1d4f7c0e3a6b9d2c"

// goldenKeyID 是 goldenModulusHex 对应模数的 kid。
//
// 这是**跨包契约的 golden**：apps/auth 的 svcoidc.DeriveKeyID 与本包 deriveKeyID
// 是同一算法的两份实现（pkg 不能反向依赖 apps，故无法共享代码），两侧测试都断言
// 这同一个字面量。改动任一侧的派生算法（如截断长度、编码方式）都会让另一侧红。
// 若这里红了但 svcoidc 侧没红，说明两侧已经漂移——kid 一旦与 OP 签发的不一致，
// 全部请求都会 401。
const goldenKeyID = "DcYoXzsKCpT1wWLw_BYXaA"

func goldenPublicKey() *rsa.PublicKey {
	n, ok := new(big.Int).SetString(goldenModulusHex, 16)
	if !ok {
		panic("goldenModulusHex is not valid hex")
	}
	return &rsa.PublicKey{N: n, E: 65537}
}

// generateRSAKey 生成测试用 2048 位 RSA 私钥。
func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// privateKeyPEM 把私钥编码为指定类型的 PEM（PKCS#8 或 PKCS#1）。
func privateKeyPEM(t *testing.T, key *rsa.PrivateKey, pkcs1 bool) string {
	t.Helper()
	var (
		der []byte
		typ string
		err error
	)
	if pkcs1 {
		der, typ = x509.MarshalPKCS1PrivateKey(key), "RSA PRIVATE KEY"
	} else {
		der, err = x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)
		typ = "PRIVATE KEY"
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
}

// newJWKSServer 起一个返回单 key JWKS 文档的测试端点。
func newJWKSServer(t *testing.T, kid string, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	})
	require.NoError(t, err)
	// 路径以 /keys 结尾：显式端点形态，SDK 不会再去推导 discovery。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newFailingJWKSServer 起一个恒返回 500 的端点（模拟 JWKS 故障）。
func newFailingJWKSServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- deriveKeyID ---------------------------------------------------------

func TestDeriveKeyID_Golden(t *testing.T) {
	// 锁定派生算法：与 apps/auth svcoidc.DeriveKeyID 共用同一 golden 字面量。
	assert.Equal(t, goldenKeyID, deriveKeyID(goldenPublicKey()))
}

func TestDeriveKeyID_StableAcrossCalls(t *testing.T) {
	key := generateRSAKey(t)
	first := deriveKeyID(&key.PublicKey)
	require.NotEmpty(t, first)
	// kid 必须稳定：同一把公钥每次派生结果一致，否则验签侧按 kid 查表必然失配。
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, deriveKeyID(&key.PublicKey))
	}
	// 不同公钥必须得到不同 kid（避免"所有 key 撞成同一个 kid"）。
	other := generateRSAKey(t)
	assert.NotEqual(t, first, deriveKeyID(&other.PublicKey))
}

func TestDeriveKeyID_Nil(t *testing.T) {
	assert.Empty(t, deriveKeyID(nil))
}

// --- parseRSAPrivateKeyPEM ----------------------------------------------

func TestParseRSAPrivateKeyPEM(t *testing.T) {
	key := generateRSAKey(t)

	t.Run("PKCS#8", func(t *testing.T) {
		parsed, err := parseRSAPrivateKeyPEM([]byte(privateKeyPEM(t, key, false)))
		require.NoError(t, err)
		assert.Equal(t, key.N, parsed.N)
	})

	t.Run("PKCS#1", func(t *testing.T) {
		parsed, err := parseRSAPrivateKeyPEM([]byte(privateKeyPEM(t, key, true)))
		require.NoError(t, err)
		assert.Equal(t, key.N, parsed.N)
	})

	t.Run("not PEM", func(t *testing.T) {
		_, err := parseRSAPrivateKeyPEM([]byte("not a pem at all"))
		assert.Error(t, err)
	})

	t.Run("wrong PEM block type", func(t *testing.T) {
		block := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("x")})
		_, err := parseRSAPrivateKeyPEM(block)
		assert.Error(t, err)
	})

	t.Run("garbage DER", func(t *testing.T) {
		block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("garbage")})
		_, err := parseRSAPrivateKeyPEM(block)
		assert.Error(t, err)
	})
}

// --- localPublicKeys ----------------------------------------------------

func TestLocalPublicKeys_KidPrecedence(t *testing.T) {
	key := generateRSAKey(t)
	pemStr := privateKeyPEM(t, key, false)
	derived := deriveKeyID(&key.PublicKey)

	t.Run("explicit item kid wins", func(t *testing.T) {
		keys := localPublicKeys(&config.Config{OIDC: config.OIDC{
			SigningKeyID:         "ignored-cfg-kid",
			SigningPrivateKeyPEM: pemStr,
			Keys:                 []config.SigningKeyConfig{{Kid: "explicit-kid", PrivateKeyPEM: pemStr, Active: true}},
		}})
		require.Len(t, keys, 1)
		_, ok := keys["explicit-kid"]
		assert.True(t, ok, "应使用 keys[].kid")
	})

	t.Run("falls back to signingKeyID", func(t *testing.T) {
		keys := localPublicKeys(&config.Config{OIDC: config.OIDC{
			SigningKeyID:         "cfg-kid",
			SigningPrivateKeyPEM: pemStr,
		}})
		require.Len(t, keys, 1)
		_, ok := keys["cfg-kid"]
		assert.True(t, ok, "未配 kid 时应回落到 oidc.signingKeyID")
	})

	t.Run("falls back to derived kid", func(t *testing.T) {
		keys := localPublicKeys(&config.Config{OIDC: config.OIDC{SigningPrivateKeyPEM: pemStr}})
		require.Len(t, keys, 1)
		_, ok := keys[derived]
		assert.True(t, ok, "都未配时应回落到 deriveKeyID")
	})
}

func TestLocalPublicKeys_MultiKeyAndPath(t *testing.T) {
	keyA, keyB := generateRSAKey(t), generateRSAKey(t)
	dir := t.TempDir()
	pathB := filepath.Join(dir, "key-b.pem")
	require.NoError(t, os.WriteFile(pathB, []byte(privateKeyPEM(t, keyB, true)), 0o600))

	keys := localPublicKeys(&config.Config{OIDC: config.OIDC{Keys: []config.SigningKeyConfig{
		{Kid: "a", PrivateKeyPEM: privateKeyPEM(t, keyA, false), Active: true},
		{Kid: "b", PrivateKeyPath: pathB},
		// 不可读路径与空配置项必须被跳过，而不是让整份密钥集失败。
		{Kid: "missing", PrivateKeyPath: filepath.Join(dir, "nope.pem")},
		{Kid: "empty"},
	}}})

	require.Len(t, keys, 2)
	assert.Equal(t, keyA.N, keys["a"].N)
	assert.Equal(t, keyB.N, keys["b"].N)
}

func TestLocalPublicKeys_InvalidPEMIsSkipped(t *testing.T) {
	keys := localPublicKeys(&config.Config{OIDC: config.OIDC{Keys: []config.SigningKeyConfig{
		{Kid: "bad", PrivateKeyPEM: "-----BEGIN PRIVATE KEY-----\nnope\n-----END PRIVATE KEY-----"},
	}}})
	assert.Empty(t, keys)
}

func TestLocalPublicKeys_NilConf(t *testing.T) {
	assert.Empty(t, localPublicKeys(nil))
}

// --- NewKeySourceFromConfig --------------------------------------------

func TestNewKeySourceFromConfig_UnavailableFailsClosed(t *testing.T) {
	// 没有任何可用来源时必须报错，绝不能返回一个"能构造但取不到键"的空来源，
	// 否则调用方会把 fail-closed 变成静默不校验。
	_, err := NewKeySourceFromConfig(nil)
	assert.Error(t, err)

	_, err = NewKeySourceFromConfig(&config.Config{})
	assert.Error(t, err)
}

func TestNewKeySourceFromConfig_JWKSURLTakesPriority(t *testing.T) {
	jwksKey, localKey := generateRSAKey(t), generateRSAKey(t)
	srv := newJWKSServer(t, "jwks-kid", &jwksKey.PublicKey)

	conf := &config.Config{OIDC: config.OIDC{
		JWKSURL: srv.URL + "/keys",
		// issuer 与本地快照同时存在：都必须让位给 JWKSURL。
		Issuer:               "http://127.0.0.1:1/oidc",
		SigningPrivateKeyPEM: privateKeyPEM(t, localKey, false),
	}}

	ks, err := NewKeySourceFromConfig(conf)
	require.NoError(t, err)

	pub, err := ks.PublicKey(context.Background(), "jwks-kid")
	require.NoError(t, err, "应能从 JWKS 端点取到公钥")
	assert.Equal(t, jwksKey.PublicKey.N, pub.N)

	// 本地快照的 kid 不应存在于来源中：证明没有落到本地兜底分支。
	_, err = ks.PublicKey(context.Background(), deriveKeyID(&localKey.PublicKey))
	assert.Error(t, err, "配置了 JWKSURL 时不应混入本地签名密钥")
}

func TestNewKeySourceFromConfig_UnreachableJWKSDoesNotFallBack(t *testing.T) {
	// 关键不变式：JWKS 端点配了但不可用时报错，**不得**静默降级到本地快照。
	// 静默降级会让"JWKS 配错/挂了"表现为"能启动但轮换后全 401"，极难排查。
	localKey := generateRSAKey(t)
	srv := newFailingJWKSServer(t)

	conf := &config.Config{OIDC: config.OIDC{
		JWKSURL:              srv.URL + "/keys",
		SigningPrivateKeyPEM: privateKeyPEM(t, localKey, false),
	}}

	_, err := NewKeySourceFromConfig(conf)
	assert.Error(t, err, "JWKS 拉取失败必须 fail-fast")
}

func TestNewKeySourceFromConfig_IssuerResolvesJWKS(t *testing.T) {
	key := generateRSAKey(t)
	jwks := newJWKSServer(t, "issuer-kid", &key.PublicKey)

	// 本仓 OP 把 JWKS 发布在 {issuer}/keys，SDK 对 issuer 形态会按
	// discovery → {issuer}/keys → {issuer}/.well-known/jwks.json 逐个尝试。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oidc/keys" {
			http.Redirect(w, r, jwks.URL+"/keys", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	ks, err := NewKeySourceFromConfig(&config.Config{OIDC: config.OIDC{Issuer: srv.URL + "/oidc"}})
	require.NoError(t, err)

	pub, err := ks.PublicKey(context.Background(), "issuer-kid")
	require.NoError(t, err)
	assert.Equal(t, key.PublicKey.N, pub.N)
}

func TestNewKeySourceFromConfig_LocalSnapshot(t *testing.T) {
	key := generateRSAKey(t)
	conf := &config.Config{OIDC: config.OIDC{
		SigningKeyID:         "snapshot-kid",
		SigningPrivateKeyPEM: privateKeyPEM(t, key, false),
	}}

	ks, err := NewKeySourceFromConfig(conf)
	require.NoError(t, err)

	pub, err := ks.PublicKey(context.Background(), "snapshot-kid")
	require.NoError(t, err)
	assert.Equal(t, key.PublicKey.N, pub.N)
}

// TestKeySourceIsSDKAlias 编译期锁定「单别名」这一决策：把 oidckit.KeySource 直接
// 交给只接受 SDK 类型的构造器，无需任何转换。
//
// 若有人把 KeySource 改成"方法集相同的另一个独立接口"，本用例立刻编译失败——
// 而那正是 back-channel logout 接收端与 access token 验签**无法共用同一个公钥来源**
// 的信号（D3 的回退条件）。
func TestKeySourceIsSDKAlias(t *testing.T) {
	var ks KeySource = rp.NewKeysFromSet(map[string]*rsa.PublicKey{"k": goldenPublicKey()})

	// 1) access token 验签路径：SDK 校验器直接吃这个来源。
	_ = rp.NewVerifier(ks)

	// 2) logout_token 解析路径：同一个来源直接传给 back-channel logout 入口。
	// token 本身是垃圾串，必然解析失败；这里验证的是**类型同一**（能编译）。
	_, err := ParseLogoutToken("not-a-jwt", ks, "https://op.example.com", "client-1")
	require.Error(t, err)
}

// --- ResolveKeySource ---------------------------------------------------

func TestResolveKeySource_InjectedWinsWithoutNetwork(t *testing.T) {
	key := generateRSAKey(t)
	injected := rp.NewKeysFromSet(map[string]*rsa.PublicKey{"injected-kid": &key.PublicKey})

	// conf 里的 JWKSURL 指向必然连不通的地址：一旦走到"按配置自建"就会失败/发网络请求。
	conf := &config.Config{OIDC: config.OIDC{JWKSURL: "http://127.0.0.1:1/keys"}}

	got, err := ResolveKeySource(injected, conf)
	require.NoError(t, err)
	assert.Equal(t, KeySource(injected), got, "注入的 KeySource 必须原样返回")

	pub, err := got.PublicKey(context.Background(), "injected-kid")
	require.NoError(t, err)
	assert.Equal(t, key.PublicKey.N, pub.N)
}

func TestResolveKeySource_NilInjectedFallsBackToConfig(t *testing.T) {
	// 未注入（独立部署）时按配置自建；配置不可用时同样 fail-closed。
	_, err := ResolveKeySource(nil, &config.Config{})
	assert.Error(t, err)

	key := generateRSAKey(t)
	conf := &config.Config{OIDC: config.OIDC{SigningKeyID: "solo", SigningPrivateKeyPEM: privateKeyPEM(t, key, false)}}
	got, err := ResolveKeySource(nil, conf)
	require.NoError(t, err)
	_, err = got.PublicKey(context.Background(), "solo")
	assert.NoError(t, err)
}
