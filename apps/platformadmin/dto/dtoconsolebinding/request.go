package dtoconsolebinding

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gobject"
)

type ConsoleBindingListReq struct {
	gobject.PageQuery
	Host     string                     `json:"host" form:"host"`
	TenantID string                     `json:"tenantId" form:"tenantId"`
	Kind     model.ConsoleBindingKind   `json:"kind" form:"kind"`
	Status   model.ConsoleBindingStatus `json:"status" form:"status"`
}

type ConsoleBindingEnableReq struct {
	BindingID string `json:"-" uri:"bindingID" binding:"required"`
	Operator  objidentity.Operator
}

type ConsoleBindingDisableReq struct {
	BindingID string `json:"-" uri:"bindingID" binding:"required"`
	Operator  objidentity.Operator
}

type ConsoleBindingItem struct {
	ID          string                     `json:"id"`
	Host        string                     `json:"host"`
	TenantID    string                     `json:"tenantId"`
	Issuer      string                     `json:"issuer"`
	ClientID    string                     `json:"clientId"`
	Audience    string                     `json:"audience"`
	RedirectURI string                     `json:"redirectUri"`
	Kind        model.ConsoleBindingKind   `json:"kind"`
	Status      model.ConsoleBindingStatus `json:"status"`
	CreatedAt   int64                      `json:"createdAt"`
	UpdatedAt   int64                      `json:"updatedAt"`
}

type ConsoleBindingListResp struct {
	List  []ConsoleBindingItem `json:"list"`
	Total int64                `json:"total"`
}
