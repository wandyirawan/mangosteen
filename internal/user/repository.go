package user

import (
	"context"
	"time"

	"mangosteen/internal/db"
)

type Repository struct {
	db *db.Queries
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database.Query()}
}

func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) {
	row, err := r.db.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUser(row), nil
}

func (r *Repository) FindActive(ctx context.Context) ([]User, error) {
	rows, err := r.db.ListActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = *toUser(row)
	}
	return users, nil
}

func (r *Repository) FindAll(ctx context.Context) ([]User, error) {
	rows, err := r.db.ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = *toUser(row)
	}
	return users, nil
}

func (r *Repository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	user, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	if role, ok := updates["role"].(string); ok {
		user.Role = role
	}
	if active, ok := updates["active"].(bool); ok {
		user.Active = active
	}
	active := int64(0)
	if user.Active {
		active = 1
	}
	return r.db.UpdateUser(ctx, db.UpdateUserParams{
		ID:           id,
		Email:        user.Email,
		PasswordHash: user.Password,
		Role:        user.Role,
		Active:      active,
		UpdatedAt:   time.Now().Format(time.RFC3339),
	})
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	return r.db.SoftDeleteUser(ctx, db.SoftDeleteUserParams{
		ID:        id,
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
}

func (r *Repository) HardDelete(ctx context.Context, id string) error {
	return r.db.HardDeleteUser(ctx, id)
}

func (r *Repository) Activate(ctx context.Context, id string) error {
	return r.db.ActivateUser(ctx, db.ActivateUserParams{
		ID:        id,
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
}

func (r *Repository) FindAttributes(ctx context.Context, userID string) (map[string]string, error) {
	rows, err := r.db.GetUserAttributes(ctx, userID)
	if err != nil {
		return nil, err
	}
	attrs := make(map[string]string, len(rows))
	for _, row := range rows {
		attrs[row.Key] = row.Value
	}
	return attrs, nil
}

func (r *Repository) SetAttributes(ctx context.Context, userID string, attrs map[string]string) error {
	if err := r.db.DeleteAllUserAttributes(ctx, userID); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	for k, v := range attrs {
		if err := r.db.SetUserAttribute(ctx, db.SetUserAttributeParams{
			UserID:    userID,
			Key:       k,
			Value:     v,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) DeleteAttribute(ctx context.Context, userID, key string) error {
	return r.db.DeleteUserAttribute(ctx, db.DeleteUserAttributeParams{
		UserID: userID,
		Key:    key,
	})
}

func (r *Repository) DeleteAllAttributes(ctx context.Context, userID string) error {
	return r.db.DeleteAllUserAttributes(ctx, userID)
}

func toUser(row db.User) *User {
	return &User{
		ID:        row.ID,
		Email:     row.Email,
		Password: row.PasswordHash,
		Role:     row.Role,
		Active:   row.Active == 1,
	}
}