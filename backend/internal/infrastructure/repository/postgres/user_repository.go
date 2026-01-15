package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implements domain.UserRepository with PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user
func (r *UserRepository) Create(user *domain.User) error {
	ctx := context.Background()

	query := `
		INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	ctx := context.Background()

	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*domain.User, error) {
	ctx := context.Background()

	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users WHERE email = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

// GetAll retrieves all users
func (r *UserRepository) GetAll() ([]*domain.User, error) {
	ctx := context.Background()

	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

// Update updates an existing user
func (r *UserRepository) Update(user *domain.User) error {
	ctx := context.Background()

	query := `
		UPDATE users 
		SET email = $1, name = $2, role = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query,
		user.Email,
		user.Name,
		user.Role,
		time.Now(),
		user.ID,
	)

	return err
}

// Delete removes a user
func (r *UserRepository) Delete(id uuid.UUID) error {
	ctx := context.Background()
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}





