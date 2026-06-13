package core_zap_logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	core_logger "github.com/rubtsov-ilya/golang-starter/internal/core/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger — обёртка над *zap.Logger, которая дополнительно хранит
// файловый дескриптор для корректного закрытия при завершении приложения.
type Logger struct {
	zapLogger *zap.Logger

	file *os.File
}

// Проверка на этапе компиляции, что структура реализует интерфейс.
var _ core_logger.Logger = (*Logger)(nil)

// NewLogger создаёт логгер, который пишет в stdout и в файл одновременно.
// Каждый запуск создаёт новый лог-файл с именем вида "2006-01-02T15-04-05.000000.log".
//
// zapcore.NewTee объединяет несколько «ядер» (outputs) в одно:
// запись в одно ядро — автоматически запись во все.
func NewLogger(config Config) (*Logger, error) {
	zapLvl := zap.NewAtomicLevel()
	if err := zapLvl.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, fmt.Errorf("unmarshal log level: %w", err)
	}

	if err := os.MkdirAll(config.Folder, 0755); err != nil {
		return nil, fmt.Errorf("mkdir log folder: %w", err)
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")
	logFilePath := filepath.Join(
		config.Folder,
		fmt.Sprintf("%s.log", timestamp),
	)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000000")

	// ConsoleEncoder — человекочитаемый формат (не JSON), удобен для разработки.
	zapEncoder := zapcore.NewConsoleEncoder(zapConfig)

	// NewTee направляет логи одновременно в stdout и в файл.
	core := zapcore.NewTee(
		zapcore.NewCore(zapEncoder, zapcore.AddSync(os.Stdout), zapLvl),
		zapcore.NewCore(zapEncoder, zapcore.AddSync(logFile), zapLvl),
	)

	// zap.AddCaller() добавляет к каждому сообщению имя файла и строку.
	zapLogger := zap.New(core, zap.AddCaller())

	return &Logger{
		zapLogger: zapLogger,
		file:      logFile,
	}, nil
}

func (l *Logger) Debug(msg string, fields ...core_logger.Field) {
	l.zapLogger.Debug(msg, toZapFields(fields...)...)
}
func (l *Logger) Info(msg string, fields ...core_logger.Field) {
	l.zapLogger.Info(msg, toZapFields(fields...)...)
}
func (l *Logger) Warn(msg string, fields ...core_logger.Field) {
	l.zapLogger.Warn(msg, toZapFields(fields...)...)
}
func (l *Logger) Error(msg string, fields ...core_logger.Field) {
	l.zapLogger.Error(msg, toZapFields(fields...)...)
}
func (l *Logger) Fatal(msg string, fields ...core_logger.Field) {
	l.zapLogger.Fatal(msg, toZapFields(fields...)...)
}
func (l *Logger) Panic(msg string, fields ...core_logger.Field) {
	l.zapLogger.Panic(msg, toZapFields(fields...)...)
}

// With создаёт дочерний логгер с дополнительными полями.
// Переопределяем метод, чтобы возвращать *core_logger.Logger (с файлом),
// а не базовый *zap.Logger.
func (l *Logger) With(field ...core_logger.Field) core_logger.Logger {
	return &Logger{
		zapLogger: l.zapLogger.With(toZapFields(field...)...),
		file:      l.file,
	}
}

// Close закрывает файл логов. Должен вызываться через defer в main().
func (l *Logger) Close() {
	if err := l.file.Close(); err != nil {
		fmt.Println("failed to close application logger:", err)
	}
}
