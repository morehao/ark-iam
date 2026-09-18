package dtoperson

import "github.com/morehao/ark-iam/pkg/model"

type PersonDetailResp struct {
	PersonID     string             `json:"personID"`
	Username     string             `json:"username"`
	PrimaryEmail string             `json:"primaryEmail"`
	PrimaryPhone string             `json:"primaryPhone"`
	Name         string             `json:"name"`
	Avatar       string             `json:"avatar"`
	Status       model.PersonStatus `json:"status"`
}
