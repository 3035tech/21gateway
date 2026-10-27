package dtos

type MFAStatusOutput struct {
	Body struct {
		Enabled bool `json:"enabled"`
	}
}
type MFABeginInput struct {
	Body struct {
		CurrentPassword string `json:"currentPassword" minLength:"1" maxLength:"128" writeOnly:"true"`
	}
}
type MFABeginOutput struct {
	Body struct {
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	}
}
type MFAConfirmInput struct {
	Body struct {
		Code string `json:"code" minLength:"6" maxLength:"6"`
	}
}
type MFAConfirmOutput struct {
	Body struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
}
type MFADisableInput struct {
	Body struct {
		CurrentPassword string `json:"currentPassword" minLength:"1" maxLength:"128" writeOnly:"true"`
		Code            string `json:"code" minLength:"6" maxLength:"32"`
	}
}
