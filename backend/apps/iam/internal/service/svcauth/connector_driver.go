package svcauth

import (
	"encoding/json"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
)

func validateOIDCConnectorConfig(config ConnectorConfig) error {
	if config.Issuer == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return code.GetError(code.ConnectorGetDetailError)
	}
	if config.Provider != connectorProviderMicrosoft {
		return nil
	}
	tenant, ok := config.Raw["tenant"]
	if !ok {
		return code.GetError(code.ConnectorGetDetailError)
	}
	tenantStr, ok := tenant.(string)
	if !ok || tenantStr == "" {
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
	DriverType() string
	ValidateConfig(config ConnectorConfig) error
	BuildAuthorizationURL(ctx *gin.Context, input *ConnectorAuthorizeInput) (*ConnectorAuthorizeOutput, error)
	ExchangeCallback(ctx *gin.Context, input *ConnectorCallbackInput) (*ConnectorCallbackOutput, error)
	TestConnection(ctx *gin.Context, input *ConnectorTestInput) (*ConnectorTestOutput, error)
}

type connectorDriverRegistry struct {
	drivers map[string]ConnectorDriver
}

func newConnectorDriverRegistry(drivers ...ConnectorDriver) *connectorDriverRegistry {
	registry := &connectorDriverRegistry{drivers: make(map[string]ConnectorDriver, len(drivers))}
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

func (r *connectorDriverRegistry) Get(driverType string) (ConnectorDriver, bool) {
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
	if connector == nil || connector.ID == 0 && connector.Protocol == "" && connector.Provider == "" && len(connector.Config) == 0 {
		return ConnectorConfig{}, code.GetError(code.ConnectorNotExistError)
	}

	config := ConnectorConfig{
		Protocol: connector.Protocol,
		Provider: connector.Provider,
	}
	if len(connector.Config) == 0 {
		config.Raw = map[string]any{}
		return config, nil
	}
	raw := make(map[string]any)
	if err := json.Unmarshal(connector.Config, &raw); err != nil {
		return ConnectorConfig{}, code.GetError(code.ConnectorGetDetailError)
	}
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return ConnectorConfig{}, code.GetError(code.ConnectorGetDetailError)
	}
	config.Protocol = connector.Protocol
	config.Provider = connector.Provider
	config.Raw = raw
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
	driverType string
}

func newStubConnectorDriver(driverType string) ConnectorDriver {
	return &stubConnectorDriver{driverType: driverType}
}

func (d *stubConnectorDriver) DriverType() string {
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
