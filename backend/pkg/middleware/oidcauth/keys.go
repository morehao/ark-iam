package oidcauth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
)

// LoadSigningPublicKey returns a closure yielding the RSA public key used to
// verify OIDC access tokens issued by the auth app. It reads the signing key
// from conf.OIDC.SigningPrivateKeyPath (file) or SigningPrivateKeyPEM.
func LoadSigningPublicKey(conf *pkgconfig.Config) func() *rsa.PublicKey {
	var publicKey *rsa.PublicKey
	if conf != nil && conf.OIDC.SigningPrivateKeyPath != "" {
		if pemData, err := os.ReadFile(conf.OIDC.SigningPrivateKeyPath); err == nil {
			if block, _ := pem.Decode(pemData); block != nil {
				if pk, err := parsePrivateKey(block.Bytes); err == nil {
					publicKey = &pk.PublicKey
				}
			}
		}
	} else if conf != nil && conf.OIDC.SigningPrivateKeyPEM != "" {
		if block, _ := pem.Decode([]byte(conf.OIDC.SigningPrivateKeyPEM)); block != nil {
			if pk, err := parsePrivateKey(block.Bytes); err == nil {
				publicKey = &pk.PublicKey
			}
		}
	}
	return func() *rsa.PublicKey { return publicKey }
}

func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if pk, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaKey, ok := pk.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(der)
}
