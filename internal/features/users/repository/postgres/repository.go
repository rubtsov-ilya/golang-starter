// Package users_postgres_repository реализует доступ к данным пользователей в PostgreSQL.
package users_postgres_repository

import (
	core_postgres_pool "github.com/rubtsov-ilya/golang-starter/internal/core/repository/postgres/pool"
)

// UsersRepository — реализация репозитория пользователей на базе PostgreSQL.
type UsersRepository struct {
	writer core_postgres_pool.Pool
	reader core_postgres_pool.Pool
}

// NewUsersRepository создаёт репозиторий пользователей с переданным пулом соединений.
func NewUsersRepository(
	writer core_postgres_pool.Pool,
	reader core_postgres_pool.Pool,
) *UsersRepository {
	return &UsersRepository{
		writer: writer,
		reader: reader,
	}
}
