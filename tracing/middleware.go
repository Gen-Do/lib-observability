package tracing

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Gen-Do/lib-obersvability/env"
)

// HTTPMiddleware создает middleware для трейсинга HTTP запросов
func HTTPMiddleware() func(http.Handler) http.Handler {
	enabled := env.GetBool("TRACING_ENABLED", false)
	if !enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пропускаем служебные эндпоинты
			if shouldSkipPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Получаем операционное имя для спана
			operationName := getOperationName(r)

			// Создаем спан
			ctx, span := StartSpan(r.Context(), operationName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethodKey.String(r.Method),
					semconv.HTTPURLKey.String(r.URL.String()),
					semconv.HTTPRouteKey.String(getRoutePath(r)),
					semconv.HTTPSchemeKey.String(r.URL.Scheme),
					attribute.String("http.host", r.Host),
					semconv.HTTPUserAgentKey.String(r.UserAgent()),
					attribute.Int("http.request_content_length", int(r.ContentLength)),
				),
			)
			defer span.End()

			// Обновляем контекст в запросе
			r = r.WithContext(ctx)

			// Создаем wrapper для ResponseWriter чтобы захватить статус код
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Выполняем следующий handler
			next.ServeHTTP(ww, r)

			// Добавляем информацию об ответе в спан
			statusCode := ww.Status()
			span.SetAttributes(
				semconv.HTTPStatusCodeKey.Int(statusCode),
				attribute.Int("http.response_content_length", int(ww.BytesWritten())),
			)

			// Устанавливаем статус спана на основе HTTP статус кода
			if statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Добавляем request ID если доступен
			if reqID := middleware.GetReqID(ctx); reqID != "" {
				span.SetAttributes(attribute.String("http.request_id", reqID))
			}
		})
	}
}

// shouldSkipPath проверяет, нужно ли пропустить путь при трейсинге
func shouldSkipPath(path string) bool {
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

// getOperationName возвращает имя операции для спана
func getOperationName(r *http.Request) string {
	route := getRoutePath(r)
	if route != "" && route != "/" {
		return r.Method + " " + route
	}
	return r.Method + " " + r.URL.Path
}

// getRoutePath пытается получить шаблон маршрута из chi роутера
func getRoutePath(r *http.Request) string {
	// Пытаемся получить паттерн маршрута из chi
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if routePattern := routeContext.RoutePattern(); routePattern != "" {
			return routePattern
		}
	}

	// Если не удалось получить паттерн, возвращаем путь как есть
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return path
}
