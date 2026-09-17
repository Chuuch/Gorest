package postgres

import (
	"context"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/files/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db:db}
}

func (r *Repository) Create(
	ctx context.Context,
	file *domain.File,
) error {
	const query = `
			INSERT INTO files (
					id,
					organization_id,
					project_id,
					uploaded_by,
					object_key,
					filename,
					content_type,
					size_bytes,
					created_at,
					updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		file.ID,
		file.OrganizationID,
		file.ProjectID,
		file.UploadedBy,
		file.ObjectKey,
		file.Filename,
		file.ContentType,
		file.Size,
		file.CreatedAt,
		file.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	return nil
}

func (r *Repository) ListByProjectID(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*domain.File, error) {
	const query = `
			SELECT
				id,
				organization_id,
				project_id,
				uploaded_by,
				object_key,
				filename,
				content_type,
				size_bytes,
				created_at,
				updated_at
			FROM files
			WHERE organization_id = $1 AND project_id = $2
			ORDER BY created_at DESC
		`

	rows, err := r.db.Query(ctx, query, organizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	files := make([]*domain.File, 0)

	for rows.Next() {
		var file domain.File

		if err := rows.Scan(
			&file.ID,
			&file.OrganizationID,
			&file.ProjectID,
			&file.UploadedBy,
			&file.ObjectKey,
			&file.Filename,
			&file.ContentType,
			&file.Size,
			&file.CreatedAt,
			&file.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}

		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	return files, nil
}
