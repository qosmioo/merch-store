package repository

import "github.com/jackc/pgx/v4"

type Transaction interface {
	pgx.Tx
}
