module github.com/morehao/ark-iam/examples/embedded-iam-demo

go 1.26.1

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/gin-gonic/gin v1.12.0
	github.com/morehao/ark-iam/iam v0.3.0
	github.com/redis/go-redis/v9 v9.18.0
)

replace (
	github.com/morehao/ark-iam/iam => ../../../iam
	github.com/morehao/ark-iam/pkg => ../../../pkg
	github.com/morehao/ark-iam/sdk => ../../../sdk
)
