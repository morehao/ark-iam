package dtoperson

type PersonDetailResp struct {
	PersonID     uint   `json:"personID"`
	Username     string `json:"username"`
	PrimaryEmail string `json:"primaryEmail"`
	PrimaryPhone string `json:"primaryPhone"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	IsSuspended  int8   `json:"isSuspended"`
}