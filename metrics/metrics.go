// Package metrics предоставляет интеграцию с Prometheus для сбора метрик
package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Gen-Do/lib-obersvability/env"
)

// Metrics структура для управления Prometheus метриками
type Metrics struct {
	registry     *prometheus.Registry
	enabled      bool
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	once         sync.Once // для lazy initialization метрик
}

// New создает новый экземпляр Metrics
func New() *Metrics {
	enabled := env.GetBool("METRICS_ENABLED", true)

	m := &Metrics{
		registry: prometheus.NewRegistry(),
		enabled:  enabled,
	}

	if !enabled {
		return m
	}

	// Регистрируем стандартные метрики Go сразу
	m.registry.MustRegister(collectors.NewGoCollector())
	m.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return m
}

// initHTTPMetrics инициализирует HTTP метрики (вызывается один раз при первом использовании middleware)
func (m *Metrics) initHTTPMetrics() {
	if !m.enabled {
		return
	}

	// Базовые лейблы для всех метрик
	baseLabels := prometheus.Labels{
		"service_name": env.GetServiceName(),
		"env_name":     string(env.GetEnvName()),
	}

	// Метрика для подсчета HTTP запросов
	m.httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests",
			ConstLabels: baseLabels,
		},
		[]string{"method", "path", "status_code"},
	)

	// Метрика для измерения длительности HTTP запросов
	m.httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request duration in seconds",
			ConstLabels: baseLabels,
			Buckets:     prometheus.DefBuckets,
		},
		[]string{"method", "path", "status_code"},
	)

	// Регистрируем метрики
	m.registry.MustRegister(m.httpRequests)
	m.registry.MustRegister(m.httpDuration)
}

// GetRegistry возвращает Prometheus registry для добавления пользовательских метрик
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// Handler возвращает HTTP handler для эндпоинта /metrics
func (m *Metrics) Handler() http.Handler {
	if !m.enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware создает middleware для сбора HTTP метрик
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	if !m.enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Инициализируем HTTP метрики только один раз при первом запросе
			// sync.Once гарантирует, что initHTTPMetrics вызовется только один раз
			m.once.Do(m.initHTTPMetrics)

			// Пропускаем служебные эндпоинты
			if m.shouldSkipPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Создаем wrapper для ResponseWriter чтобы захватить статус код
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Выполняем следующий handler
			next.ServeHTTP(ww, r)

			// Записываем метрики (они уже инициализированы благодаря sync.Once)
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(ww.Status())

			// Получаем путь из роута chi, если доступен
			path := m.getRoutePath(r)

			// Увеличиваем счетчик запросов
			m.httpRequests.WithLabelValues(r.Method, path, statusCode).Inc()

			// Записываем время выполнения
			m.httpDuration.WithLabelValues(r.Method, path, statusCode).Observe(duration)
		})
	}
}

// shouldSkipPath проверяет, нужно ли пропустить путь при сборе метрик
func (m *Metrics) shouldSkipPath(path string) bool {
	skipPaths := []string{
		"/metrics",
		"/health",
		"/healthz",
		"/ready",
		"/readiness",
		"/liveness",
		"/ping",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// getRoutePath пытается получить шаблон маршрута из chi роутера
func (m *Metrics) getRoutePath(r *http.Request) string {
	// Пытаемся получить паттерн маршрута из chi
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if routePattern := routeContext.RoutePattern(); routePattern != "" {
			return routePattern
		}
	}

	// Если не удалось получить паттерн, возвращаем путь как есть
	// но обрезаем query параметры
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return path
}
