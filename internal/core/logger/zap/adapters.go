package core_zap_logger

import (
	core_logger "github.com/rubtsov-ilya/golang-starter/internal/core/logger"
	"go.uber.org/zap"
)

func toZapFields(fields ...core_logger.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		// Важно: если передан error с ключом "error", используем специальный zap.Error
		// для корректного сохранения стек-трейса ошибки.
		if err, ok := f.Value.(error); ok && f.Key == "error" {
			zapFields[i] = zap.Error(err)
		} else {
			zapFields[i] = zap.Any(f.Key, f.Value)
		}
	}
	return zapFields
}
