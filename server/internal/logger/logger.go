// Package logger предоставляет абстракцию над logging библиотекой zap.
//
// Реализует интерфейс Logger с поддержкой уровней логирования (Debug, Info, Warn, Error),
// структурированных полей (zap.Field) и graceful shutdown через Sync().
//
// Использование:
//
//	log, err := logger.New(cfg)
//	if err != nil {
//	    // обработка ошибки
//	}
//	defer log.Sync()
//
//	log.Info("server started", zap.String("port", cfg.Server.Port))
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gophkeeper/server/internal/config"
)

// Logger определяет интерфейс для структурированного логирования.
//
// Поддерживаемые уровни:
//   - Debug: отладочная информация (только в development режиме)
//   - Info:  нормальные события (запуск, остановка, успешные операции)
//   - Warn:  предупреждения (потенциальные проблемы)
//   - Error: ошибки (требующие внимания, но не фатальные)
//   - Fatal: фатальные ошибки (вызовет os.Exit(1) после записи лога)
//
// Метод With создаёт дочерний логгер с предустановленными полями.
// Метод Sync сбрасывает буферы (вызывается при завершении программы).
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
}

// ZapLogger реализует интерфейс Logger поверх go.uber.org/zap.
//
// Особенности:
//   - В debug режиме используется DevelopmentConfig (цветной вывод, стектрейсы)
//   - В release режиме используется ProductionConfig (JSON-формат, оптимизация)
type ZapLogger struct {
	logger *zap.Logger
}

// New создаёт новый экземпляр Logger на основе конфигурации.
//
// Алгоритм:
//  1. Если cfg.Server.Debug == true → DevelopmentConfig
//  2. Иначе → ProductionConfig
//  3. Добавляет timestamp в формате ISO8601
//
// Параметры:
//   - cfg: конфигурация сервера (определяет режим debug/production)
//
// Возвращает:
//   - Logger: готовый к использованию логгер
//   - error: ошибка при инициализации zap (например, неверная конфигурация)
func New(cfg *config.Config) (Logger, error) {
	// Выбор конфигурации в зависимости от режима
	var zapCfg zap.Config

	if cfg.Server.Debug {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// Сборка логгера
	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	return &ZapLogger{logger: zapLogger}, nil
}

// Debug логирует сообщение на уровне Debug.
// Используется для отладочной информации, не выводится в production режиме.
func (l *ZapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Info логирует сообщение на уровне Info.
// Используется для нормальных событий (запуск, остановка, успешные операции).
func (l *ZapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Warn логирует сообщение на уровне Warn.
// Используется для предупреждений (потенциальные проблемы, fallback механизмы).
func (l *ZapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// Error логирует сообщение на уровне Error.
// Используется для ошибок, которые требуют внимания, но не останавливают программу.
func (l *ZapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

// Fatal логирует сообщение на уровне Fatal и завершает программу с кодом 1.
//
// Выполняет:
//  1. Запись лога с уровнем Fatal
//  2. Вызов os.Exit(1)
//
// Используется для критических ошибок, при которых продолжение работы невозможно:
//   - Невалидная конфигурация
//   - Невозможность подключения к БД
//   - Отсутствие обязательных секретов
func (l *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

// With создаёт дочерний логгер с предустановленными полями.
//
// Использование:
//
//	requestLogger := log.With(zap.String("request_id", reqID))
//	requestLogger.Info("processing request")
func (l *ZapLogger) With(fields ...zap.Field) Logger {
	return &ZapLogger{logger: l.logger.With(fields...)}
}

// Sync сбрасывает буферизованные логи в вывод.
// Должен вызываться перед завершением программы (defer log.Sync()).
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}
