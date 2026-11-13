package svcperson

import (
	"context"
	"github.com/morehao/ark-iam/backend/apps/auth/dto/dtoperson"
)

type PersonProfileSvc interface {
	Detail(ctx context.Context, req *dtoperson.PersonDetailReq) (*dtoperson.PersonDetailResp, error)
	UpdatePassword(ctx context.Context, req *dtoperson.PersonUpdatePasswordReq) error
}
