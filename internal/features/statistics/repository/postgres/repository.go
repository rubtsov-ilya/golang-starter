// Package statistics_postgres_repository реализует доступ к данным для расчёта статистики.
// Репозиторий читает задачи с опциональной фильтрацией — доменная логика подсчёта
// статистики находится в domain.CreateStatistics, не здесь.
package statistics_postgres_repository

import core_postgres_pool "github.com/rubtsov-ilya/golang-starter/internal/core/repository/postgres/pool"

// StatisticsRepository — репозиторий для получения данных, необходимых для статистики.
type StatisticsRepository struct {
	writer core_postgres_pool.Pool
	reader core_postgres_pool.Pool
}

// NewStatisticsRepository создаёт репозиторий статистики с переданными пулами соединений.
func NewStatisticsRepository(
	writer core_postgres_pool.Pool,
	reader core_postgres_pool.Pool,
) *StatisticsRepository {
	return &StatisticsRepository{
		writer: writer,
		reader: reader,
	}
}
