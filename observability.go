// Package observability предоставляет единую точку входа для всех компонентов observability
package observability

import (
	"context"
	"fmt"

	"github.com/Gen-Do/lib-observability/env"
	"github.com/Gen-Do/lib-observability/logger"
	"github.com/Gen-Do/lib-observability/metrics"
	"github.com/Gen-Do/lib-observability/tracing"
)

// Observability центральная структура для управления всеми компонентами observability
type Observability struct {
	logger  logger.Logger
	metrics *metrics.Metrics
	tracing bool // флаг, указывающий на то, что трейсинг инициализирован
}

// New создает новый экземпляр Observability с инициализацией всех компонентов
// Инициализирует логгер, метрики и трейсинг на основе переменных окружения
func New(ctx context.Context) (*Observability, error) {
	obs := &Observability{}

	env.LoadEnvFiles()

	// Инициализация логгера
	obs.logger = logger.New(ctx)
	obs.logger.Info(ctx, "Logger initialized")

	// Инициализация метрик
	obs.metrics = metrics.New()
	obs.logger.Info(ctx, "Metrics initialized")

	// Инициализация трейсинга
	if err := tracing.New(); err != nil {
		obs.logger.Error(obs.logger.WithError(ctx, err), "Failed to initialize tracing")
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	obs.tracing = true
	obs.logger.Info(ctx, "Tracing initialized")

	obs.logger.Info(ctx, "Observability stack initialized successfully")
	return obs, nil
}

// MustNew создает новый экземпляр Observability или паникует при ошибке
// Удобно для случаев, когда ошибка инициализации критична
func MustNew(ctx context.Context) *Observability {
	obs, err := New(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize observability: %v", err))
	}
	return obs
}

// GetLogger возвращает настроенный логгер
func (o *Observability) GetLogger() logger.Logger {
	return o.logger
}

// GetMetrics возвращает настроенные метрики
func (o *Observability) GetMetrics() *metrics.Metrics {
	return o.metrics
}

// IsTracingEnabled проверяет, инициализирован ли трейсинг
func (o *Observability) IsTracingEnabled() bool {
	return o.tracing
}

// Shutdown корректно завершает работу всех компонентов observability
func (o *Observability) Shutdown(ctx context.Context) error {
	o.logger.Info(ctx, "Shutting down observability stack")

	// Завершаем работу трейсинга
	if o.tracing {
		if err := tracing.Shutdown(ctx); err != nil {
			o.logger.Error(o.logger.WithError(ctx, err), "Failed to shutdown tracing")
			return fmt.Errorf("failed to shutdown tracing: %w", err)
		}
		o.logger.Info(ctx, "Tracing shutdown completed")
	}

	o.logger.Info(ctx, "Observability stack shutdown completed")
	return nil
}

// Logger возвращает логгер (краткий алиас для GetLogger)
func (o *Observability) Logger() logger.Logger {
	return o.logger
}

// Metrics возвращает метрики (краткий алиас для GetMetrics)
func (o *Observability) Metrics() *metrics.Metrics {
	return o.metrics
}
