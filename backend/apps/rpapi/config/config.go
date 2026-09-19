// Package config 承载 rpapi 应用的运行时配置。
package config

import (
	"os"
	"path/filepath"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/golib/gutil"
)

var Conf *pkgconfig.Config

func InitConf() {
	LoadConfig(getConfigPath())
}

func LoadConfig(configPath string) {
	var cfg pkgconfig.Config
	gutil.LoadYamlConfig(configPath, &cfg)
	Conf = &cfg
}

func getConfigPath() string {
	if configPath := os.Getenv("APP_CONFIG_PATH"); configPath != "" {
		return configPath
	}
	relativePath := "../config/config.yaml"
	if fileExists(relativePath) {
		return relativePath
	}
	execPath, err := os.Executable()
	if err == nil {
		absPath := filepath.Join(filepath.Dir(execPath), "..", "config", "config.yaml")
		if fileExists(absPath) {
			return absPath
		}
	}
	return relativePath
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
