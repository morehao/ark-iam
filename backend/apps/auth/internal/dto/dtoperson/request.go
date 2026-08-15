package dtoperson

type PersonDetailReq struct {
}

type PersonUpdatePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}
