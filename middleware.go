package observability

import (
	"net/http"

	"github.com/Gen-Do/lib-obersvability/env"
	"github.com/Gen-Do/lib-obersvability/logger"
	"github.com/Gen-Do/lib-obersvability/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// HTTPMiddleware возвращает набор middleware для HTTP сервера
// Включает логирование, метрики и трейсинг
func (o *Observability) HTTPMiddleware() []func(http.Handler) http.Handler {
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

// RegisterMetricsRoute регистрирует эндпоинт /metrics в указанном роутере
func (o *Observability) RegisterMetricsRoute(router interface{}) {
	switch r := router.(type) {
	case *chi.Mux:
		r.Handle("/metrics", o.MetricsHandler())
	case chi.Router:
		r.Handle("/metrics", o.MetricsHandler())
	}
}

// RegisterHealthRoute регистрирует эндпоинт /health в указанном роутере
func (o *Observability) RegisterHealthRoute(router interface{}) {
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"` + env.GetServiceName() + `","version":"` + env.GetServiceVersion() + `"}`))
	})

	switch r := router.(type) {
	case *chi.Mux:
		r.Handle("/health", healthHandler)
		r.Handle("/healthz", healthHandler) // Kubernetes style
	case chi.Router:
		r.Handle("/health", healthHandler)
		r.Handle("/healthz", healthHandler)
	}
}

// RegisterObservabilityRoutes регистрирует все служебные эндпоинты
func (o *Observability) RegisterObservabilityRoutes(router interface{}) {
	o.RegisterMetricsRoute(router)
	o.RegisterHealthRoute(router)
}

// SetupRouter полностью настраивает роутер с middleware и служебными эндпоинтами
func (o *Observability) SetupRouter(router *chi.Mux) {
	// Добавляем базовые middleware
	router.Use(middleware.RequestID)

	// Добавляем observability middleware
	for _, mw := range o.HTTPMiddleware() {
		router.Use(mw)
	}

	// Регистрируем служебные эндпоинты
	o.RegisterObservabilityRoutes(router)
}

// NewRouter создает новый роутер с полной настройкой observability
func (o *Observability) NewRouter() *chi.Mux {
	r := chi.NewRouter()
	o.SetupRouter(r)
	return r
}
