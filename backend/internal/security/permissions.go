package security

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

type Resource string
type Action string

const (
	ResourceCompanies     Resource = "companies"
	ResourceUsers         Resource = "users"
	ResourceCredentials   Resource = "credentials"
	ResourceMessages      Resource = "messages"
	ResourceCallbacks     Resource = "callbacks"
	ResourceAudit         Resource = "audit"
	ResourceNotifications Resource = "notifications"
	ActionRead            Action   = "read"
	ActionWrite           Action   = "write"
)

type Permissions struct{ enforcer *casbin.Enforcer }

func NewPermissions() (*Permissions, error) {
	policy, err := model.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act`)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(policy)
	if err != nil {
		return nil, err
	}
	for _, resource := range []Resource{ResourceCompanies, ResourceCredentials, ResourceMessages, ResourceCallbacks} {
		if _, err := enforcer.AddPolicy(string(models.RoleViewer), string(resource), string(ActionRead)); err != nil {
			return nil, err
		}
	}
	for _, resource := range []Resource{ResourceCompanies, ResourceCredentials, ResourceCallbacks} {
		if _, err := enforcer.AddPolicy(string(models.RoleOperator), string(resource), string(ActionWrite)); err != nil {
			return nil, err
		}
	}
	for _, action := range []Action{ActionRead, ActionWrite} {
		if _, err := enforcer.AddPolicy(string(models.RoleAdmin), string(ResourceUsers), string(action)); err != nil {
			return nil, err
		}
	}
	if _, err := enforcer.AddPolicy(string(models.RoleAdmin), string(ResourceAudit), string(ActionRead)); err != nil {
		return nil, err
	}
	for _, action := range []Action{ActionRead, ActionWrite} {
		if _, err := enforcer.AddPolicy(string(models.RoleAdmin), string(ResourceNotifications), string(action)); err != nil {
			return nil, err
		}
	}
	if _, err := enforcer.AddPolicy(string(models.RoleAdmin), string(ResourceMessages), string(ActionWrite)); err != nil {
		return nil, err
	}
	if _, err := enforcer.AddGroupingPolicy(string(models.RoleAdmin), string(models.RoleOperator)); err != nil {
		return nil, err
	}
	if _, err := enforcer.AddGroupingPolicy(string(models.RoleOperator), string(models.RoleViewer)); err != nil {
		return nil, err
	}
	return &Permissions{enforcer: enforcer}, nil
}
func (p *Permissions) Allow(role models.Role, resource Resource, action Action) bool {
	allowed, err := p.enforcer.Enforce(string(role), string(resource), string(action))
	return err == nil && allowed
}
