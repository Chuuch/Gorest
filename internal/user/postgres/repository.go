package postgres

import "github.com/jackc/pgx/v5/pgxpool"

type UserRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}
