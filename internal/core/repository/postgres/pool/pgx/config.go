package core_pgx_pool

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// NodeConfig описывает параметры подключения к конкретному инстансу (узлу) базы данных.
type NodeConfig struct {
	Host     string        `envconfig:"HOST"     required:"true"`
	Port     string        `envconfig:"PORT"     default:"5432"`
	User     string        `envconfig:"USER"     required:"true"`
	Password string        `envconfig:"PASSWORD" required:"true"`
	Database string        `envconfig:"DB"       required:"true"`
	Timeout  time.Duration `envconfig:"TIMEOUT"  required:"true"`
	MaxConns int32         `envconfig:"MAX_CONNS" default:"20"`
}

// Config объединяет настройки для Master и Replica.
type Config struct {
	Master  NodeConfig `envconfig:"MASTER"`
	Replica NodeConfig `envconfig:"REPLICA"`
}

// NewConfig читает конфигурацию пула из переменных окружения.
func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("POSTGRES", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

// NewConfigMust — «Must»-вариант конструктора: паникует при ошибке.
func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get Postgres connection pool config: %w", err)
		panic(err)
	}

	return config
}
