package config

import (
	"github.com/morehao/golib/dbaccess/dbes"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/protocol/ghttp"
)

type Config struct {
	Server      Server                    `yaml:"server"`
	Log         map[string]glog.LogConfig `yaml:"log"`
	Trace       gtrace.TraceConfig        `yaml:"trace"`
	DBConfigs   []dbgorm.Config           `yaml:"db_configs"`
	RedisConfig dbredis.RedisConfig       `yaml:"redis_config"`
	ESConfigs   []dbes.ESConfig           `yaml:"es_configs"`
	Client      Client                    `yaml:"client"`
	JWT         JWT                       `yaml:"jwt"`
	OIDC        OIDC                      `yaml:"oidc"`
	Password    PasswordConfig            `yaml:"password"`
	Security    SecurityConfig            `yaml:"security"`
	MasterKey   string                    `yaml:"masterKey"`
}

type SecurityConfig struct {
	Login LoginGuardConfig `yaml:"login"`
}

type LoginGuardConfig struct {
	MaxFailures int `yaml:"maxFailures"`
	WindowSec   int `yaml:"windowSec"`
	LockSec     int `yaml:"lockSec"`
}

type OIDC struct {
	Issuer                string `yaml:"issuer"`
	FrontendLoginURL      string `yaml:"frontendLoginURL"`
	CookieDomain          string `yaml:"cookieDomain"`
	SigningKeyID          string `yaml:"signingKeyID"`
	SigningPrivateKeyPath string `yaml:"signingPrivateKeyPath"`
	SigningPrivateKeyPEM  string `yaml:"signingPrivateKeyPEM"`
	EncryptionKey         string `yaml:"encryptionKey"`
	EncryptionKeyID       string `yaml:"encryptionKeyID"`
	AllowInsecure         bool   `yaml:"allowInsecure"`
	AuthRequestTTL        int    `yaml:"authRequestTTL"`
	AuthCodeTTL           int    `yaml:"authCodeTTL"`
	SpentCodeTTL          int    `yaml:"spentCodeTTL"`
	SessionTTL            int    `yaml:"sessionTTL"`
}

func (o OIDC) SSOCookieDomain() string {
	return o.CookieDomain
}

type PasswordConfig struct {
	Prefix string `yaml:"prefix"`
}

type JWT struct {
	SignKey string `yaml:"signKey"`
}

type Server struct {
	Name string `yaml:"name"` // 服务名称
	Port string `yaml:"port"` // 服务端口
	Env  string `yaml:"env"`  // 环境变量
}

type Client struct {
	HTTPBingo *ghttp.Client `yaml:"httpbingo"`
}
