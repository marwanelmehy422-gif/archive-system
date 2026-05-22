package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"archive-system/internal/database"
	"archive-system/internal/models"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUsername - جيب يوزر بالـ username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, organization_id, username, email, password_hash,
		       full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE username = $1 AND is_active = TRUE`

	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetByUsername: %w", err)
	}
	return user, nil
}

// GetByEmail - جيب يوزر بالـ email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, organization_id, username, email, password_hash,
		       full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = TRUE`

	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetByEmail: %w", err)
	}
	return user, nil
}

// GetByID - جيب يوزر بالـ ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, organization_id, username, email, password_hash,
		       full_name, role, is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = TRUE`

	user := &models.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	return user, nil
}

// Create - عمل يوزر جديد
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (organization_id, username, email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query,
		user.OrganizationID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("Create user: %w", err)
	}
	return nil
}

// UpdateLastLogin - حدّث وقت آخر دخول
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE users SET last_login_at = $1 WHERE id = $2`,
		now, userID,
	)
	return err
}

// GetOrganizationByCode - جيب organization بالـ code
func (r *UserRepository) GetOrganizationByCode(ctx context.Context, code string) (*models.Organization, error) {
	query := `
		SELECT id, name, code, is_active, created_at, updated_at
		FROM organizations
		WHERE code = $1 AND is_active = TRUE`

	org := &models.Organization{}
	err := r.db.Pool.QueryRow(ctx, query, code).Scan(
		&org.ID,
		&org.Name,
		&org.Code,
		&org.IsActive,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetOrganizationByCode: %w", err)
	}
	return org, nil
}
