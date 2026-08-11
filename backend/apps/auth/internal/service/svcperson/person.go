package svcperson

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoperson"
)

type PersonProfileSvc interface {
	Detail(ctx *gin.Context, req *dtoperson.PersonDetailReq) (*dtoperson.PersonDetailResp, error)
	UpdatePassword(ctx *gin.Context, req *dtoperson.PersonUpdatePasswordReq) error
}
