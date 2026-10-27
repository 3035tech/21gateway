package models

type CredentialCiphertext struct {
	ID                     uint64
	Username               string
	ProviderTokenEncrypted string
}
type CompanyCiphertext struct {
	ID                     uint64
	ProviderTokenEncrypted string
}
type MOCiphertext struct {
	ID               uint64
	CredentialID     uint64
	EventKey         string
	EncryptedMessage string
}
type RotationResult struct {
	Scanned   int64 `json:"scanned"`
	Eligible  int64 `json:"eligible"`
	Changed   int64 `json:"changed"`
	Conflicts int64 `json:"conflicts"`
}
