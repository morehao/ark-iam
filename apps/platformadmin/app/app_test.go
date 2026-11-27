package app

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	padconfig "github.com/morehao/ark-iam/apps/platformadmin/config"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/golib/dbaccess/dbgorm"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"
)

func TestStandalonePlatformMountsConnectorManagement(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kid": "test",
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	op := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	defer op.Close()

	previous := padconfig.Conf
	t.Cleanup(func() { padconfig.Conf = previous })
	padconfig.Conf = &pkgconfig.Config{
		Server:    pkgconfig.Server{Env: "dev"},
		DBConfigs: []dbgorm.Config{{URL: "sqlite://:memory:"}},
		OIDC:      pkgconfig.OIDC{Issuer: op.URL, JWKSURL: op.URL + "/jwks.json"},
	}
	engine := gin.New()
	app, err := Routers(engine)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	found := false
	for _, route := range engine.Routes() {
		switch route.Path {
		case "/v1/platform/connectors":
			if route.Method == http.MethodGet {
				found = true
			}
		case "/v1/auth/connectors/callback", "/v1/tenant/roles":
			t.Errorf("unexpected route in platform process: %s", route.Path)
		}
	}
	if !found {
		t.Fatal("standalone platform process did not mount connector management")
	}
}
