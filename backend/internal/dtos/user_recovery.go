package dtos

import "github.com/21mobile/sms-gateway/internal/models"

type RecoveryRequestInput struct {
	Body struct {
		Email string `json:"email" format:"email" maxLength:"254"`
	}
}
type RecoveryResetInput struct {
	Body struct {
		Token       string `json:"token" minLength:"64" maxLength:"64" writeOnly:"true"`
		Factor      string `json:"factor,omitempty" maxLength:"32" writeOnly:"true"`
		NewPassword string `json:"newPassword" minLength:"15" maxLength:"128" writeOnly:"true"`
	}
}
type InviteUserInput struct {
	Body struct {
		Name   string      `json:"name" minLength:"1" maxLength:"120"`
		Email  string      `json:"email" format:"email" maxLength:"254"`
		Role   models.Role `json:"role" enum:"admin,operator,viewer"`
		Active bool        `json:"active"`
	}
}
