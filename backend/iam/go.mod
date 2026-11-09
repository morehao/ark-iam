module github.com/morehao/ark-iam/iam

go 1.26.1

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/morehao/ark-iam/pkg v0.0.0
	github.com/morehao/ark-iam/sdk v0.0.0
	github.com/morehao/golib v1.33.0
	github.com/redis/go-redis/v9 v9.18.0
	gorm.io/gorm v1.31.1
)

replace (
	github.com/morehao/ark-iam/pkg => ../pkg
	github.com/morehao/ark-iam/sdk => ../../sdk
)
