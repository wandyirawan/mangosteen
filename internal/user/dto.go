package user

type UpdateUserDTO struct {
	Email string `json:"email" validate:"omitempty,email"`
	Role  string `json:"role" validate:"omitempty,oneof=admin user"`
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}