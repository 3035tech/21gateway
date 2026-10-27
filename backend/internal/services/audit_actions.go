package services

const (
	auditCreated           = "created"
	auditUpdated           = "updated"
	auditArchived          = "archived"
	auditRotated           = "password_rotated"
	auditLogin             = "login"
	auditMFAEnabled        = "mfa_enabled"
	auditMFADisabled       = "mfa_disabled"
	auditInviteAccepted    = "user_invite_accepted"
	auditPasswordRecovered = "password_recovered"
	initialSessionVersion  = 1
)
