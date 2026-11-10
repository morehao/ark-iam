module github.com/morehao/ark-iam/sdk

go 1.26.1

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/elastic/elastic-transport-go/v8 v8.10.0 // indirect
	github.com/elastic/go-elasticsearch/v8 v8.19.3 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/gin-contrib/sse v1.1.1 // indirect
	github.com/gin-gonic/gin v1.12.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/morehao/golib v1.33.0 // indirect
	github.com/pelletier/go-toml/v2 v2.3.0 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/redis/go-redis/v9 v9.18.0 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.31.1 // indirect
)

require (
	github.com/morehao/ark-iam/auth v0.0.0
	github.com/morehao/ark-iam/pkg v0.0.0
	github.com/morehao/ark-iam/platformadmin v0.0.0
	github.com/morehao/ark-iam/tenantadmin v0.0.0
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/morehao/ark-iam/pkg => ../pkg

replace github.com/morehao/ark-iam/auth => ../backend/apps/auth

replace github.com/morehao/ark-iam/platformadmin => ../backend/apps/platformadmin

replace github.com/morehao/ark-iam/tenantadmin => ../backend/apps/tenantadmin
