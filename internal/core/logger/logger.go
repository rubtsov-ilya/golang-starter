package core_logger

import (
	"context"
)

// Field — абстрактное представление поля лога, независимое от внешних библиотек.
type Field struct {
	Key   string
	Value any
}

// Хелперы для создания полей (заменяют вызовы zap.String, zap.Error и т.д.):
func String(key string, val string) Field {
	return Field{Key: key, Value: val}
}
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}
func Any(key string, val any) Field {
	return Field{Key: key, Value: val}
}

// Logger — чистый интерфейс логгера.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Panic(msg string, fields ...Field)
	With(fields ...Field) Logger
	Close()
}

// loggerContextKey — приватный тип ключа для context.WithValue.
// Использование отдельного типа (а не string) исключает коллизии ключей
// с другими пакетами, которые тоже хранят данные в контексте.
type loggerContextKey struct{}

var (
	key = loggerContextKey{}
)

// ToContext сохраняет интерфейс Logger в контексте.
func ToContext(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, key, log)
}

// FromContext извлекает интерфейс Logger из контекста.
func FromContext(ctx context.Context) Logger {
	log, ok := ctx.Value(key).(Logger)
	if !ok {
		panic("no logger in context")
	}
	return log
}
