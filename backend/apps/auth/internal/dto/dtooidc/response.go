package dtooidc

type OIDCLoginResp struct {
	ContinueURL string `json:"continueURL"`
	SessionID   string `json:"sessionID,omitempty"`
}
