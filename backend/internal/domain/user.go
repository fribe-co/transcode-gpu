package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user permission level
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
	UserRoleViewer   UserRole = "viewer"
)

// User represents a user entity
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewUser creates a new user
func NewUser(email, name string, role UserRole) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// HasPermission checks if user has required permission
func (u *User) HasPermission(requiredRole UserRole) bool {
	roleHierarchy := map[UserRole]int{
		UserRoleViewer:   1,
		UserRoleOperator: 2,
		UserRoleAdmin:    3,
	}
	return roleHierarchy[u.Role] >= roleHierarchy[requiredRole]
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	Create(user *User) error
	GetByID(id uuid.UUID) (*User, error)
	GetByEmail(email string) (*User, error)
	GetAll() ([]*User, error)
	Update(user *User) error
	Delete(id uuid.UUID) error
}





