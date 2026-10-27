package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type UserView struct {
	ID        uint64      `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	Role      models.Role `json:"role" enum:"admin,operator,viewer"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"createdAt"`
}
type UserBody struct {
	Name     string      `json:"name" minLength:"1" maxLength:"120"`
	Email    string      `json:"email" format:"email" maxLength:"254"`
	Role     models.Role `json:"role" enum:"admin,operator,viewer"`
	Active   bool        `json:"active"`
	Password string      `json:"password,omitempty" minLength:"15" maxLength:"128"`
}
type CreateUserInput struct{ Body UserBody }
type UpdateUserInput struct {
	IDInput
	Body UserBody
}
type UserOutput struct{ Body UserView }
type UsersOutput struct{ Body Page[UserView] }
type ChangeOwnPasswordBody struct {
	CurrentPassword string `json:"currentPassword" minLength:"1" maxLength:"128" writeOnly:"true"`
	NewPassword     string `json:"newPassword" minLength:"15" maxLength:"128" writeOnly:"true"`
}
type ChangeOwnPasswordInput struct{ Body ChangeOwnPasswordBody }

func UserDTO(value models.SystemUser) UserView {
	return UserView{ID: value.ID, Name: value.Name, Email: value.Email, Role: value.Role, Active: value.Active, CreatedAt: value.CreatedAt}
}
