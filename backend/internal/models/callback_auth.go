package models

type CallbackAuthMode string

const (
	CallbackAuthNone   CallbackAuthMode = "none"
	CallbackAuthHeader CallbackAuthMode = "header"
	CallbackAuthBasic  CallbackAuthMode = "basic"
)

// CallbackAuth is stored per destination. The encrypted secret is never sent
// to the administrative API or written to callback logs.
type CallbackAuth struct {
	Mode            CallbackAuthMode `json:"mode"`
	HeaderName      string           `json:"headerName,omitempty"`
	Username        string           `json:"username,omitempty"`
	SecretEncrypted string           `json:"secretEncrypted,omitempty"`
}
