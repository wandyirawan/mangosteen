package user

type UpdateUserDTO struct {
	Email string `json:"email" validate:"omitempty,email"`
	Role  string `json:"role" validate:"omitempty,oneof=admin user"`
}

type User struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Password   string            `json:"password,omitempty"`
	Role       string            `json:"role"`
	Active     bool              `json:"active"`
	Attributes map[string]string `json:"attributes,omitempty"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

type SetAttributesDTO struct {
	Attributes map[string]string `json:"attributes"`
}