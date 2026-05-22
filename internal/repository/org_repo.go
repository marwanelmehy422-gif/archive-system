package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"archive-system/internal/database"
	"archive-system/internal/models"
)

type OrgRepository struct {
	db *database.DB
}

func NewOrgRepository(db *database.DB) *OrgRepository {
	return &OrgRepository{db: db}
}

func (r *OrgRepository) Create(ctx context.Context, org *models.Organization) error {
	query := `
		INSERT INTO organizations (name, code)
		VALUES ($1, $2)
		RETURNING id, is_active, created_at, updated_at`

	err := r.db.Pool.QueryRow(ctx, query, org.Name, org.Code).Scan(
		&org.ID, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create org: %w", err)
	}
	return nil
}

func (r *OrgRepository) GetAll(ctx context.Context) ([]models.Organization, error) {
	query := `SELECT id, name, code, is_active, created_at, updated_at FROM organizations ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all orgs: %w", err)
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Code, &org.IsActive, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (r *OrgRepository) GetByCode(ctx context.Context, code string) (*models.Organization, error) {
	query := `SELECT id, name, code, is_active, created_at, updated_at FROM organizations WHERE code = $1`

	org := &models.Organization{}
	err := r.db.Pool.QueryRow(ctx, query, code).Scan(
		&org.ID, &org.Name, &org.Code, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org by code: %w", err)
	}
	return org, nil
}

func (r *OrgRepository) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	query := `SELECT id, name, code, is_active, created_at, updated_at FROM organizations WHERE id = $1`

	org := &models.Organization{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Code, &org.IsActive, &org.CreatedAt, &org.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org by id: %w", err)
	}
	return org, nil
}
