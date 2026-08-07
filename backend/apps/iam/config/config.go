package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/morehao/golib/dbaccess/dbes"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/gutil"
	"github.com/morehao/golib/protocol/ghttp"
)

var Conf *Config

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

// InitConf 初始化配置（运行时使用，自动查找配置文件）
func InitConf() {
	configPath := getConfigPath()
	LoadConfig(configPath)
}

// LoadConfig 从指定路径加载配置并设置全局变量（测试环境可直接使用）
func LoadConfig(configPath string) {
	fmt.Println("Load config file:", configPath)

	var cfg Config
	gutil.LoadYamlConfig(configPath, &cfg)
	Conf = &cfg
}

// getConfigPath 获取配置文件路径，优先级：环境变量 > 相对路径 > 绝对路径
func getConfigPath() string {
	// 优先使用环境变量
	if configPath := os.Getenv("APP_CONFIG_PATH"); configPath != "" {
		return configPath
	}

	// 尝试相对路径（从当前工作目录）
	relativePath := "../config/config.yaml"
	if fileExists(relativePath) {
		return relativePath
	}

	// 获取可执行文件所在目录的上级目录
	execPath, err := os.Executable()
	if err == nil {
		// 可执行文件目录的上级目录/config/config.yaml
		absPath := filepath.Join(filepath.Dir(execPath), "..", "config", "config.yaml")
		if fileExists(absPath) {
			return absPath
		}
	}

	// 默认返回相对路径
	return relativePath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
