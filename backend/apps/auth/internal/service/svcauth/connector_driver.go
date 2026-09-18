package svcauth

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
)

func validateOIDCConnectorConfig(config ConnectorConfig) error {
	if config.Issuer == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	if config.Provider != connectorProviderMicrosoft {
		return nil
	}
	if config.Tenant == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	return nil
}

func validateOAuth2ConnectorConfig(config ConnectorConfig) error {
	if config.AuthURL == "" || config.TokenURL == "" || config.UserInfoURL == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	return nil
}

func resolveConnectorRedirectURI(input *ConnectorAuthorizeInput) string {
	if input == nil {
		return ""
	}
	if input.RedirectURI != "" {
		return input.RedirectURI
	}
	return input.Config.RedirectURI
}

type ConnectorDriver interface {
	DriverType() model.ConnectorProtocol
	ValidateConfig(config ConnectorConfig) error
	BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error)
	ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error)
	TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error)
}

type connectorDriverRegistry struct {
	drivers map[model.ConnectorProtocol]ConnectorDriver
}

func newConnectorDriverRegistry(drivers ...ConnectorDriver) *connectorDriverRegistry {
	registry := &connectorDriverRegistry{drivers: make(map[model.ConnectorProtocol]ConnectorDriver, len(drivers))}
	for _, driver := range drivers {
		if driver == nil {
			continue
		}
		registry.drivers[driver.DriverType()] = driver
	}
	return registry
}

func defaultConnectorDriverRegistry() *connectorDriverRegistry {
	return newConnectorDriverRegistry(
		NewOIDCDriver(),
		NewOAuth2Driver(),
	)
}

func (r *connectorDriverRegistry) Get(driverType model.ConnectorProtocol) (ConnectorDriver, bool) {
	if r == nil {
		return nil, false
	}
	driver, ok := r.drivers[driverType]
	return driver, ok
}

func (r *connectorDriverRegistry) Select(config ConnectorConfig) (ConnectorDriver, bool) {
	return r.Get(config.Protocol)
}

func buildConnectorConfig(connector *model.ConnectorEntity) (ConnectorConfig, error) {
	if connector == nil || connector.ID == "" && connector.Protocol == "" && connector.Provider == "" {
		return ConnectorConfig{}, code.GetError(code.ConnectorNotExistError)
	}

	config := connector.Config
	config.Protocol = connector.Protocol
	config.Provider = connector.Provider
	return config, nil
}

func selectDriverForConnector(registry *connectorDriverRegistry, connector *model.ConnectorEntity) (ConnectorDriver, ConnectorConfig, error) {
	config, err := buildConnectorConfig(connector)
	if err != nil {
		return nil, ConnectorConfig{}, err
	}
	driver, ok := registry.Select(config)
	if !ok {
		return nil, ConnectorConfig{}, code.GetError(code.ConnectorGetDetailError)
	}
	if err := driver.ValidateConfig(config); err != nil {
		return nil, ConnectorConfig{}, err
	}
	return driver, config, nil
}

type stubConnectorDriver struct {
	driverType model.ConnectorProtocol
}

func newStubConnectorDriver(driverType model.ConnectorProtocol) ConnectorDriver {
	return &stubConnectorDriver{driverType: driverType}
}

func (d *stubConnectorDriver) DriverType() model.ConnectorProtocol {
	return d.driverType
}

func (d *stubConnectorDriver) ValidateConfig(config ConnectorConfig) error {
	switch d.driverType {
	case connectorDriverTypeOIDC:
		return validateOIDCConnectorConfig(config)
	case connectorDriverTypeOAuth2:
		return validateOAuth2ConnectorConfig(config)
	default:
		return code.GetError(code.ConnectorGetDetailError)
	}
}

func (d *stubConnectorDriver) BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error) {
	_ = ctx
	params := url.Values{}
	params.Set("redirect_uri", resolveConnectorRedirectURI(input))
	params.Set("state", input.State)
	return &ConnectorAuthorizeOutput{
		AuthorizationURL: "https://authorization.url/oauth/authorize?" + params.Encode(),
		Nonce:            "",
	}, nil
}

func (d *stubConnectorDriver) ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error) {
	return &ConnectorCallbackOutput{}, nil
}

func (d *stubConnectorDriver) TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error) {
	return &ConnectorTestOutput{
		Success: true,
		Message: "连接成功",
	}, nil
}
