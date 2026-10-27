package controllers

import (
	"context"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"time"
)

const sessionUserID = "user_id"
const sessionVersion = "session_version"
const sessionRemember = "remember"
const sessionLastActivity = "last_activity"
const adminSessionScheme = "adminSession"

type userContextKey struct{}
type SessionAccess interface {
	SessionUser(context.Context, uint64, int64) (models.SystemUser, error)
	Allowed(models.Role, security.Resource, security.Action) bool
}
type SessionGuard struct {
	users    SessionAccess
	sessions *scs.SessionManager
}

func NewSessionGuard(users SessionAccess, sessions *scs.SessionManager) *SessionGuard {
	return &SessionGuard{users: users, sessions: sessions}
}
func (c *SessionGuard) authorize(api huma.API, resource security.Resource, action security.Action) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		last := c.sessions.GetInt64(ctx.Context(), sessionLastActivity)
		idle := security.RegularSessionIdleTimeout
		if c.sessions.GetBool(ctx.Context(), sessionRemember) {
			idle = security.RememberedSessionIdleTimeout
		}
		if last != 0 && time.Since(time.Unix(last, 0)) > idle {
			_ = c.sessions.Destroy(ctx.Context())
			_ = huma.WriteErr(api, ctx, 401, "")
			return
		}
		user, err := c.users.SessionUser(ctx.Context(), uint64(c.sessions.GetInt64(ctx.Context(), sessionUserID)), c.sessions.GetInt64(ctx.Context(), sessionVersion))
		if err == nil && resource != "" && !c.users.Allowed(user.Role, resource, action) {
			err = apperrors.Wrap(apperrors.CodeForbidden, nil)
		}
		if err != nil {
			public := HTTPError(err)
			_ = huma.WriteErr(api, ctx, public.GetStatus(), "")
			return
		}
		c.sessions.Put(ctx.Context(), sessionLastActivity, time.Now().Unix())
		requestCtx := context.WithValue(ctx.Context(), userContextKey{}, user)
		if user.Role != models.RoleAdmin {
			requestCtx = security.WithCompanyScope(requestCtx, user.CompanyIDs)
		}
		next(huma.WithContext(ctx, requestCtx))
	}
}
func actor(ctx context.Context) uint64 { return ctx.Value(userContextKey{}).(models.SystemUser).ID }
func (c *SessionGuard) operation(api huma.API, id, method, path, summary string, resource security.Resource, action security.Action) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: "/admin" + path, Summary: summary, Tags: []string{"Backoffice"}, Errors: []int{401, 403, 404, 409, 422, 429, 503}, Security: []map[string][]string{{adminSessionScheme: {}}}, Middlewares: huma.Middlewares{c.authorize(api, resource, action)}}
}
