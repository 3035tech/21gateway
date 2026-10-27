package dtos

type LoginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" maxLength:"254"`
		Password string `json:"password" minLength:"1" maxLength:"128"`
		Factor   string `json:"factor,omitempty" maxLength:"32"`
		Remember bool   `json:"remember,omitempty"`
	}
}
