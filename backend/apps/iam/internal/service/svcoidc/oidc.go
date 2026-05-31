package svcoidc

import (
	"crypto/sha256"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

const (
	pathLoggedOut = "/logged-out"
)

type OIDCProvider struct {
	Provider *op.Provider
	Storage  *OIDCStorage
}

func SetupOIDCProvider(issuer string) (*OIDCProvider, error) {
	storage := NewOIDCStorage()

	key := sha256.Sum256([]byte("test-key-32-bytes-for-oidc-encrp"))
	keyId := "enc-key-1"

	config := &op.Config{
		CryptoKey:                key,
		CryptoKeyId:              keyId,
		DefaultLogoutRedirectURI: pathLoggedOut,
		CodeMethodS256:           true,
		AuthMethodPost:           true,
		GrantTypeRefreshToken:    true,
		SupportedUILocales:       []language.Tag{language.Chinese, language.English},
		DeviceAuthorization: op.DeviceAuthorizationConfig{
			Lifetime:     5 * time.Minute,
			PollInterval: 5 * time.Second,
			UserFormPath: "/device",
			UserCode:     op.UserCodeBase20,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	provider, err := op.NewOpenIDProvider(issuer, config, storage,
		op.WithAllowInsecure(),
		op.WithCustomAuthEndpoint(op.NewEndpoint("authorize")),
		op.WithLogger(logger.WithGroup("op")),
	)
	if err != nil {
		return nil, err
	}

	return &OIDCProvider{
		Provider: provider,
		Storage:  storage,
	}, nil
}

func (p *OIDCProvider) GetProviderHTTPHandler() http.Handler {
	return p.Provider
}
