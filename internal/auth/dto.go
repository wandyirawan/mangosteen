package auth

// Request DTOs
type LoginDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type RegisterDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// Response DTOs
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Type         string `json:"type"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// Domain Model
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	Active       int64
	CreatedAt    string
	UpdatedAt    string
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt string
	Revoked   int64
	CreatedAt string
}
