package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/ticketfiles/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	file *domain.File,
) error {
	const query = `
			INSERT INTO ticket_files (
				id,
				organization_id,
				ticket_id,
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
		file.TicketID,
		file.UploadedBy,
		file.ObjectKey,
		file.Filename,
		file.ContentType,
		file.Size,
		file.CreatedAt,
		file.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create ticket file: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.File, error) {
	const query = `
			SELECT
				id,
				organization_id,
				ticket_id,
				uploaded_by,
				object_key,
				filename,
				content_type,
				size_bytes,
				created_at,
				updated_at
			FROM ticket_files
			WHERE id = $1 AND organization_id = $2
		`

	var file domain.File

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	).Scan(
		&file.ID,
		&file.OrganizationID,
		&file.TicketID,
		&file.UploadedBy,
		&file.ObjectKey,
		&file.Filename,
		&file.ContentType,
		&file.Size,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFileNotFound
		}
		return nil, fmt.Errorf("get ticket file: %w", err)
	}
	return &file, nil
}

func (r *Repository) ListByTicketID(
	ctx context.Context,
	organizationID, ticketID uuid.UUID,
) ([]*domain.File, error) {
	const query = `
			SELECT
				id,
				organization_id,
				ticket_id,
				uploaded_by,
				object_key,
				filename,
				content_type,
				size_bytes,
				created_at,
				updated_at
			FROM ticket_files
			WHERE organization_id = $1 AND ticket_id = $2
			ORDER BY created_at DESC
		`

	rows, err := r.db.Query(ctx, query, organizationID, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list ticket files: %w", err)
	}
	defer rows.Close()

	files := make([]*domain.File, 0)

	for rows.Next() {
		var file domain.File

		if err := rows.Scan(
			&file.ID,
			&file.OrganizationID,
			&file.TicketID,
			&file.UploadedBy,
			&file.ObjectKey,
			&file.Filename,
			&file.ContentType,
			&file.Size,
			&file.CreatedAt,
			&file.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ticket file: %w", err)
		}

		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list ticket files: %w", err)
	}

	return files, nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	const query = `
			DELETE FROM ticket_files
			WHERE id = $1 AND organization_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		id,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("delete ticket file: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrFileNotFound
	}

	return nil
}
