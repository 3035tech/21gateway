package security

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"testing"
)

func TestPermissionsEnforceInternalRoles(t *testing.T) {
	permissions, err := NewPermissions()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		role     models.Role
		resource Resource
		action   Action
		allowed  bool
	}{
		{models.RoleViewer, ResourceCompanies, ActionRead, true},
		{models.RoleViewer, ResourceCredentials, ActionWrite, false},
		{models.RoleOperator, ResourceCredentials, ActionWrite, true},
		{models.RoleOperator, ResourceUsers, ActionWrite, false},
		{models.RoleAdmin, ResourceUsers, ActionWrite, true},
		{models.RoleAdmin, ResourceCredentials, ActionWrite, true},
		{models.RoleAdmin, ResourceAudit, ActionRead, true},
		{models.Role("external_api"), ResourceCompanies, ActionRead, false},
		{models.RoleAdmin, ResourceMessages, ActionWrite, true},
		{models.RoleOperator, ResourceMessages, ActionWrite, false},
		{models.RoleAdmin, ResourceNotifications, ActionRead, true},
		{models.RoleAdmin, ResourceNotifications, ActionWrite, true},
		{models.RoleOperator, ResourceNotifications, ActionRead, false},
		{models.RoleViewer, ResourceNotifications, ActionRead, false},
		{models.RoleViewer, ResourceOperationCases, ActionRead, true},
		{models.RoleViewer, ResourceOperationCases, ActionWrite, false},
		{models.RoleOperator, ResourceOperationCases, ActionWrite, true},
		{models.RoleAdmin, ResourceOperationCases, ActionWrite, true},
	}
	for _, c := range cases {
		if got := permissions.Allow(c.role, c.resource, c.action); got != c.allowed {
			t.Errorf("%s %s %s: allowed=%v, want %v", c.role, c.resource, c.action, got, c.allowed)
		}
	}
}
