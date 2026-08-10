package router

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/morehao/ark-iam/tenantadmin/config"
)

var OIDCPublicKey *rsa.PublicKey

// InitOIDC loads the OIDC signing public key from config so this app can validate
// OIDC access tokens issued by the auth app. tenantadmin hosts neither the OIDC
// provider nor SSO liveness endpoints (those are auth-exclusive).
func InitOIDC() func() *rsa.PublicKey {
	if appConfig := config.Conf; appConfig != nil && appConfig.OIDC.SigningPrivateKeyPath != "" {
		if pemData, err := os.ReadFile(appConfig.OIDC.SigningPrivateKeyPath); err == nil {
			if block, _ := pem.Decode(pemData); block != nil {
				if pk, err := parsePrivateKey(block.Bytes); err == nil {
					OIDCPublicKey = &pk.PublicKey
				}
			}
		}
	} else if appConfig != nil && appConfig.OIDC.SigningPrivateKeyPEM != "" {
		if block, _ := pem.Decode([]byte(appConfig.OIDC.SigningPrivateKeyPEM)); block != nil {
			if pk, err := parsePrivateKey(block.Bytes); err == nil {
				OIDCPublicKey = &pk.PublicKey
			}
		}
	}
	return func() *rsa.PublicKey { return OIDCPublicKey }
}

func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if pk, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaKey, ok := pk.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(der)
}
