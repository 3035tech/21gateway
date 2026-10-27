package models

import "time"

type UserPasswordReset struct {
	UserID    uint64 `gorm:"primaryKey"`
	TokenHash []byte
	Purpose   string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (UserPasswordReset) TableName() string { return "user_password_resets" }
