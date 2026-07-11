// Package logger предоставляет абстракцию для логирования с поддержкой контекста
package logger

import (
	"context"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/Gen-Do/lib-observability/env"
)

// Fields представляет набор полей для логирования
type Fields map[string]any

// Logger интерфейс для логирования с поддержкой контекста
type Logger interface {
	// WithField добавляет поле в контекст для последующего логирования
	WithField(ctx context.Context, key string, value any) context.Context
	// WithFields добавляет несколько полей в контекст для последующего логирования
	WithFields(ctx context.Context, fields Fields) context.Context
	// WithError добавляет ошибку в контекст для последующего логирования
	WithError(ctx context.Context, err error) context.Context

	// Debug логирует сообщение уровня Debug
	Debug(ctx context.Context, args ...any)
	// Info логирует сообщение уровня Info
	Info(ctx context.Context, args ...any)
	// Print логирует сообщение уровня Info (алиас для Info)
	Print(ctx context.Context, args ...any)
	// Warn логирует сообщение уровня Warning
	Warn(ctx context.Context, args ...any)
	// Error логирует сообщение уровня Error
	Error(ctx context.Context, args ...any)
	// Fatal логирует сообщение уровня Fatal и завершает программу
	Fatal(ctx context.Context, args ...any)
	// Panic логирует сообщение уровня Panic и вызывает панику
	Panic(ctx context.Context, args ...any)
}

// contextKey тип для ключей контекста
type contextKey string

const (
	// fieldsContextKey ключ для хранения полей логирования в контексте
	fieldsContextKey contextKey = "logger_fields"
)

// logrusAdapter адаптер для logrus логгера
type logrusAdapter struct {
	logger *logrus.Logger
}

// New создает новый экземпляр логгера на основе logrus
func New(ctx context.Context) Logger {
	logger := logrus.New()

	// Настройка уровня логирования
	logLevel := env.GetString("LOG_LEVEL", "debug")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.DebugLevel
	}
	logger.SetLevel(level)

	// Настройка формата логов
	logFormat := env.GetString("LOG_FORMAT", "json")
	switch strings.ToLower(logFormat) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	return &logrusAdapter{logger: logger}
}

// WithField добавляет поле в контекст
func (l *logrusAdapter) WithField(ctx context.Context, key string, value interface{}) context.Context {
	return l.WithFields(ctx, Fields{key: value})
}

// WithFields добавляет несколько полей в контекст.
//
// Поля родительского контекста копируются в новую map, а не дописываются на
// месте. Мутация общей map ломала два инварианта: (1) горутины, порождённые из
// одного контекста, конкурентно звали WithField/логировали → `fatal error:
// concurrent map iteration and map write` (getLogrusEntry итерирует ту же map);
// (2) поля дочернего контекста протекали обратно в родительский, потому что оба
// держали один объект map. Копия на каждую деривацию делает map неизменяемой
// после сохранения — безопасно и для конкурентного чтения, и для чтения-записи.
func (l *logrusAdapter) WithFields(ctx context.Context, newFields Fields) context.Context {
	existing := l.getFieldsFromContext(ctx)
	merged := make(Fields, len(existing)+len(newFields))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range newFields {
		merged[k] = v
	}
	return context.WithValue(ctx, fieldsContextKey, merged)
}

// WithError добавляет ошибку в контекст
func (l *logrusAdapter) WithError(ctx context.Context, err error) context.Context {
	return l.WithField(ctx, "error", err.Error())
}

// getFieldsFromContext извлекает поля из контекста
func (l *logrusAdapter) getFieldsFromContext(ctx context.Context) Fields {
	if fields, ok := ctx.Value(fieldsContextKey).(Fields); ok {
		return fields
	}
	return nil
}

// getLogrusEntry создает logrus entry с полями из контекста и базовыми полями сервиса
func (l *logrusAdapter) getLogrusEntry(ctx context.Context) *logrus.Entry {
	entry := l.logger.WithFields(logrus.Fields{
		"service_name":    env.GetServiceName(),
		"service_version": env.GetServiceVersion(),
		"env_name":        env.GetEnvName(),
	})

	// Добавляем поля из контекста
	if fields := l.getFieldsFromContext(ctx); fields != nil {
		for k, v := range fields {
			entry = entry.WithField(k, v)
		}
	}

	// Автоматически добавляем trace_id и span_id из активного OpenTelemetry span.
	// Это позволяет Grafana строить ссылки Loki → Tempo по полю trace_id.
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		entry = entry.WithField("trace_id", sc.TraceID().String())
		entry = entry.WithField("span_id", sc.SpanID().String())
	}

	return entry
}

// Debug логирует сообщение уровня Debug
func (l *logrusAdapter) Debug(ctx context.Context, args ...interface{}) {
	l.getLogrusEntry(ctx).Debug(args...)
}

// Info логирует сообщение уровня Info
func (l *logrusAdapter) Info(ctx context.Context, args ...interface{}) {
	l.getLogrusEntry(ctx).Info(args...)
}

// Print логирует сообщение уровня Info (алиас для Info)
func (l *logrusAdapter) Print(ctx context.Context, args ...interface{}) {
	l.Info(ctx, args...)
}

// Warn логирует сообщение уровня Warning
func (l *logrusAdapter) Warn(ctx context.Context, args ...interface{}) {
	l.getLogrusEntry(ctx).Warn(args...)
}

// Error логирует сообщение уровня Error
func (l *logrusAdapter) Error(ctx context.Context, args ...interface{}) {
	l.getLogrusEntry(ctx).Error(args...)
}

// Fatal логирует сообщение уровня Fatal и завершает программу
func (l *logrusAdapter) Fatal(ctx context.Context, args ...interface{}) {
	l.getLogrusEntry(ctx).Fatal(args...)
}

// Panic логирует сообщение уровня Panic со stacktrace и вызывает панику
func (l *logrusAdapter) Panic(ctx context.Context, args ...interface{}) {
	ctx = l.WithField(ctx, "stacktrace", string(debug.Stack()))
	l.getLogrusEntry(ctx).Panic(args...)
}

// Recover перехватывает панику, логирует её со stacktrace в JSON формате
// и при необходимости повторно вызывает панику (reraise=true) или подавляет её.
// Предназначен для использования через defer в main() или goroutine-обёртках.
//
// Пример:
//
//	defer logger.Recover(ctx, log, true)
func Recover(ctx context.Context, log Logger, reraise bool) {
	if rvr := recover(); rvr != nil {
		ctx = log.WithFields(ctx, Fields{
			"panic":      rvr,
			"stacktrace": string(debug.Stack()),
		})
		log.Error(ctx, "panic recovered")
		if reraise {
			panic(rvr)
		}
	}
}
