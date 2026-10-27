package repositories

import (
	"context"
	"gorm.io/gorm"
	"time"
)

const authenticationQueryTimeout = 2 * time.Second

type AuthenticationAttempts struct{ db *gorm.DB }

func NewAuthenticationAttempts(db *gorm.DB) *AuthenticationAttempts {
	return &AuthenticationAttempts{db: db}
}

// Reserve is atomic across replicas. Counts saturate to avoid overflow under abuse.
func (r *AuthenticationAttempts) Reserve(ctx context.Context, key string, limit int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, authenticationQueryTimeout)
	defer cancel()
	var count int
	err := r.db.WithContext(ctx).Raw(`INSERT INTO login_limits(key,window_start,attempts) VALUES (?,date_trunc('minute',NOW()),1) ON CONFLICT(key) DO UPDATE SET attempts=CASE WHEN login_limits.window_start=EXCLUDED.window_start THEN LEAST(login_limits.attempts+1,?) ELSE 1 END,window_start=EXCLUDED.window_start RETURNING attempts`, key, limit+1).Scan(&count).Error
	return count <= limit, err
}
func (r *AuthenticationAttempts) Blocked(ctx context.Context, key string, limit int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, authenticationQueryTimeout)
	defer cancel()
	var count int64
	err := r.db.WithContext(ctx).Table("login_limits").Where("key = ? AND window_start = date_trunc('minute',NOW()) AND attempts >= ?", key, limit).Count(&count).Error
	return count != 0, err
}
func (r *AuthenticationAttempts) Failure(ctx context.Context, key string, limit int) error {
	_, err := r.Reserve(ctx, key, limit)
	return err
}
