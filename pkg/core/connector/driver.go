package connector

import (
	"context"
	"net/url"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
)

func validateOIDCConfig(config Config) error {
	if config.Issuer == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	if config.Provider != ProviderMicrosoft {
		return nil
	}
	if config.Tenant == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	return nil
}

func validateOAuth2Config(config Config) error {
	if config.AuthURL == "" || config.TokenURL == "" || config.UserInfoURL == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	return nil
}

func resolveAuthorizeRedirectURI(input *AuthorizeInput) string {
	if input == nil {
		return ""
	}
	if input.RedirectURI != "" {
		return input.RedirectURI
	}
	return input.Config.RedirectURI
}

func resolveCallbackRedirectURI(input *CallbackInput) string {
	if input == nil {
		return ""
	}
	if input.RedirectURI != "" {
		return input.RedirectURI
	}
	return input.Config.RedirectURI
}

// Driver 连接器协议驱动（OIDC / OAuth2）。
type Driver interface {
	DriverType() model.ConnectorProtocol
	ValidateConfig(config Config) error
	BuildAuthorizationURL(ctx context.Context, input *AuthorizeInput) (*AuthorizeOutput, error)
	ExchangeCallback(ctx context.Context, input *CallbackInput) (*CallbackOutput, error)
	TestConnection(ctx context.Context, input *TestInput) (*TestOutput, error)
}

// DriverRegistry 按协议类型注册/选择驱动。
type DriverRegistry struct {
	drivers map[model.ConnectorProtocol]Driver
}

func NewDriverRegistry(drivers ...Driver) *DriverRegistry {
	registry := &DriverRegistry{drivers: make(map[model.ConnectorProtocol]Driver, len(drivers))}
	for _, driver := range drivers {
		if driver == nil {
			continue
		}
		registry.drivers[driver.DriverType()] = driver
	}
	return registry
}

// DefaultDriverRegistry 内置驱动注册表（OIDC + OAuth2）。
func DefaultDriverRegistry() *DriverRegistry {
	return NewDriverRegistry(NewOIDCDriver(), NewOAuth2Driver())
}

func (r *DriverRegistry) Get(driverType model.ConnectorProtocol) (Driver, bool) {
	if r == nil {
		return nil, false
	}
	driver, ok := r.drivers[driverType]
	return driver, ok
}

func (r *DriverRegistry) Select(config Config) (Driver, bool) {
	return r.Get(config.Protocol)
}

// BuildConfig 从连接器实体组装驱动配置（合并 protocol/provider 列到 config）。
func BuildConfig(entity *model.ConnectorEntity) (Config, error) {
	if entity == nil || entity.ID == "" && entity.Protocol == "" && entity.Provider == "" {
		return Config{}, code.GetError(code.ConnectorNotExistError)
	}
	config := entity.Config
	config.Protocol = entity.Protocol
	config.Provider = entity.Provider
	return config, nil
}

// SelectDriverForConnector 为连接器实体选择驱动并校验配置。
func SelectDriverForConnector(registry *DriverRegistry, entity *model.ConnectorEntity) (Driver, Config, error) {
	config, err := BuildConfig(entity)
	if err != nil {
		return nil, Config{}, err
	}
	driver, ok := registry.Select(config)
	if !ok {
		return nil, Config{}, code.GetError(code.ConnectorGetDetailError)
	}
	if err := driver.ValidateConfig(config); err != nil {
		return nil, Config{}, err
	}
	return driver, config, nil
}

type stubDriver struct {
	driverType model.ConnectorProtocol
}

// NewStubDriver 测试用桩驱动：不触达外部网络，仅用于单元测试注册表选择。
func NewStubDriver(driverType model.ConnectorProtocol) Driver {
	return &stubDriver{driverType: driverType}
}

func (d *stubDriver) DriverType() model.ConnectorProtocol { return d.driverType }

func (d *stubDriver) ValidateConfig(config Config) error {
	switch d.driverType {
	case DriverTypeOIDC:
		return validateOIDCConfig(config)
	case DriverTypeOAuth2:
		return validateOAuth2Config(config)
	default:
		return code.GetError(code.ConnectorGetDetailError)
	}
}

func (d *stubDriver) BuildAuthorizationURL(ctx context.Context, input *AuthorizeInput) (*AuthorizeOutput, error) {
	_ = ctx
	params := url.Values{}
	params.Set("redirect_uri", resolveAuthorizeRedirectURI(input))
	params.Set("state", input.State)
	return &AuthorizeOutput{AuthorizationURL: "https://authorization.url/oauth/authorize?" + params.Encode()}, nil
}

func (d *stubDriver) ExchangeCallback(ctx context.Context, input *CallbackInput) (*CallbackOutput, error) {
	return &CallbackOutput{}, nil
}

func (d *stubDriver) TestConnection(ctx context.Context, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Success: true, Message: "连接成功"}, nil
}
