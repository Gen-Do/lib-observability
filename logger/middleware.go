package logger

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Gen-Do/lib-observability/env"
	httputil "github.com/Gen-Do/lib-observability/internal/http"
)

// HTTPMiddleware создает middleware для логирования HTTP запросов
// Middleware можно отключить через переменную окружения LOG_HTTP_ENABLED=false
func HTTPMiddleware(logger Logger) func(http.Handler) http.Handler {
	enabled := env.GetBool("LOG_HTTP_ENABLED", true)
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

			ctx := r.Context()
			start := time.Now()

			// Создаем wrapper для ResponseWriter чтобы захватить статус код
			rw := httputil.NewResponseWriter(w)

			// Добавляем информацию о запросе в контекст логгера
			ctx = logger.WithFields(ctx, Fields{
				"http_method":      r.Method,
				"http_url":         r.URL.String(),
				"http_path":        r.URL.Path,
				"http_remote_addr": r.RemoteAddr,
				"http_user_agent":  r.UserAgent(),
				"http_request_id":  getRequestID(ctx),
			})

			// Обновляем контекст в запросе
			r = r.WithContext(ctx)

			// Выполняем следующий handler
			next.ServeHTTP(rw, r)

			// Логируем результат
			duration := time.Since(start)

			ctx = logger.WithFields(ctx, Fields{
				"http_status":        rw.Status(),
				"http_duration_ms":   duration.Milliseconds(),
				"http_bytes_written": rw.BytesWritten(),
			})

			// Определяем уровень логирования на основе статус кода
			statusCode := rw.Status()
			switch {
			case statusCode >= http.StatusInternalServerError:
				logger.Error(ctx, "HTTP request completed with server error")
			case statusCode >= http.StatusBadRequest:
				logger.Warn(ctx, "HTTP request completed with client error")
			default:
				logger.Info(ctx, "HTTP request completed")
			}
		})
	}
}

// RecovererMiddleware восстанавливается после паники и логирует её
func RecovererMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					ctx := r.Context()

					// Логируем панику как ошибку со stacktrace
					ctx = logger.WithFields(ctx, Fields{
						"panic":            rvr,
						"stacktrace":       string(debug.Stack()),
						"http_method":      r.Method,
						"http_url":         r.URL.String(),
						"http_path":        r.URL.Path,
						"http_remote_addr": r.RemoteAddr,
						"http_request_id":  getRequestID(ctx),
					})

					logger.Error(ctx, "HTTP request panicked")

					// Отправляем 500 ошибку
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// getRequestID пытается получить request ID из контекста
// Совместимо с chi middleware.RequestID
func getRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value("RequestID").(string); ok {
		return reqID
	}
	// Альтернативные ключи для совместимости
	if reqID, ok := ctx.Value("request-id").(string); ok {
		return reqID
	}
	if reqID, ok := ctx.Value("X-Request-ID").(string); ok {
		return reqID
	}
	return ""
}

// skipPrefixes — пути, пропускаемые по префиксу (включая все сабпути)
var skipPrefixes = []string{"/metrics"}

// skipSuffixes — пути, пропускаемые по суффиксу (покрывает /v1/health, /api/v1/ready и т.д.)
var skipSuffixesLog = []string{
	"/health",
	"/healthz",
	"/ready",
	"/readiness",
	"/liveness",
	"/ping",
}

// shouldSkipPath проверяет, нужно ли пропустить путь при логировании.
func shouldSkipPath(path string) bool {
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, suffix := range skipSuffixesLog {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}
