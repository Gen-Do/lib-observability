package observability

import (
	"net/http"

	"github.com/Gen-Do/lib-observability/env"
	"github.com/Gen-Do/lib-observability/logger"
	"github.com/Gen-Do/lib-observability/tracing"
)

// HTTPMiddleware возвращает единый middleware, объединяющий все компоненты observability
// Включает восстановление после паники, логирование, метрики и трейсинг в правильном порядке
func (o *Observability) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Применяем middleware в правильном порядке (изнутри наружу)
		handler := next

		// 4. Трейсинг (если включен) - самый внутренний
		if o.tracing {
			handler = tracing.HTTPMiddleware()(handler)
		}

		// 3. Метрики
		handler = o.metrics.Middleware()(handler)

		// 2. Логирование
		handler = logger.HTTPMiddleware(o.logger)(handler)

		// 1. Восстановление после паники - самый внешний
		handler = logger.RecovererMiddleware(o.logger)(handler)

		return handler
	}
}

// HTTPMiddlewares возвращает набор middleware по отдельности (для ручной настройки)
// Deprecated: используйте HTTPMiddleware() для получения единого middleware
func (o *Observability) HTTPMiddlewares() []func(http.Handler) http.Handler {
	var middlewares []func(http.Handler) http.Handler

	// Добавляем middleware для восстановления после паники
	middlewares = append(middlewares, logger.RecovererMiddleware(o.logger))

	// Добавляем middleware для логирования
	middlewares = append(middlewares, logger.HTTPMiddleware(o.logger))

	// Добавляем middleware для метрик
	middlewares = append(middlewares, o.metrics.Middleware())

	// Добавляем middleware для трейсинга (если включен)
	if o.tracing {
		middlewares = append(middlewares, tracing.HTTPMiddleware())
	}

	return middlewares
}

// LoggingMiddleware возвращает middleware для логирования HTTP запросов
func (o *Observability) LoggingMiddleware() func(http.Handler) http.Handler {
	return logger.HTTPMiddleware(o.logger)
}

// MetricsMiddleware возвращает middleware для сбора метрик HTTP запросов
func (o *Observability) MetricsMiddleware() func(http.Handler) http.Handler {
	return o.metrics.Middleware()
}

// TracingMiddleware возвращает middleware для трейсинга HTTP запросов
func (o *Observability) TracingMiddleware() func(http.Handler) http.Handler {
	return tracing.HTTPMiddleware()
}

// RecoveryMiddleware возвращает middleware для восстановления после паники
func (o *Observability) RecoveryMiddleware() func(http.Handler) http.Handler {
	return logger.RecovererMiddleware(o.logger)
}

// MetricsHandler возвращает HTTP handler для эндпоинта /metrics
func (o *Observability) MetricsHandler() http.Handler {
	return o.metrics.Handler()
}

// HealthHandler возвращает HTTP handler для health check эндпоинта
func (o *Observability) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"` + env.GetServiceName() + `","version":"` + env.GetServiceVersion() + `"}`))
	})
}

// RouterRegistrar интерфейс для роутеров, поддерживающих регистрацию HTTP handlers
type RouterRegistrar interface {
	Handle(pattern string, handler http.Handler)
}

// RegisterRoutes регистрирует все служебные эндпоинты в роутере
// Поддерживает любые роутеры, реализующие интерфейс RouterRegistrar
func (o *Observability) RegisterRoutes(router RouterRegistrar) {
	router.Handle("/metrics", o.MetricsHandler())
	router.Handle("/health", o.HealthHandler())
	router.Handle("/healthz", o.HealthHandler()) // Kubernetes style
}

// SetupHTTP полностью настраивает HTTP роутер с middleware и служебными эндпоинтами
// Принимает интерфейс HTTPRouter для максимальной совместимости
type HTTPRouter interface {
	RouterRegistrar
	Use(middlewares ...func(http.Handler) http.Handler)
}

// SetupHTTP настраивает роутер с observability middleware и служебными эндпоинтами
func (o *Observability) SetupHTTP(router HTTPRouter) {
	// Добавляем observability middleware
	router.Use(o.HTTPMiddleware())

	// Регистрируем служебные эндпоинты
	o.RegisterRoutes(router)
}
