// Package tasks_postgres_repository реализует доступ к данным задач в PostgreSQL.
// Каждая операция (GetTask, GetTasks, SaveTask, UpdateTask, DeleteTask) вынесена
// в отдельный файл для читаемости.
package tasks_postgres_repository

import core_postgres_pool "github.com/rubtsov-ilya/golang-starter/internal/core/repository/postgres/pool"

// TasksRepository — реализация репозитория задач на базе PostgreSQL.
// Принимает интерфейс core_postgres_pool.Pool, а не конкретный тип pgx,
// что позволяет подменить реализацию БД в тестах без изменения этого кода.
type TasksRepository struct {
	writer core_postgres_pool.Pool
	reader core_postgres_pool.Pool
}

// NewTasksRepository создаёт репозиторий задач с переданными пулами соединений.
func NewTasksRepository(
	writer core_postgres_pool.Pool,
	reader core_postgres_pool.Pool,
) *TasksRepository {
	return &TasksRepository{
		writer: writer,
		reader: reader,
	}
}
