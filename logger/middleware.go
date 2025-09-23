package logger

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/Gen-Do/lib-obersvability/env"
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
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Добавляем информацию о запросе в контекст логгера
			ctx = logger.WithFields(ctx, Fields{
				"http_method":      r.Method,
				"http_url":         r.URL.String(),
				"http_path":        r.URL.Path,
				"http_remote_addr": r.RemoteAddr,
				"http_user_agent":  r.UserAgent(),
				"http_request_id":  middleware.GetReqID(ctx),
			})

			// Обновляем контекст в запросе
			r = r.WithContext(ctx)

			// Выполняем следующий handler
			next.ServeHTTP(ww, r)

			// Логируем результат
			duration := time.Since(start)

			ctx = logger.WithFields(ctx, Fields{
				"http_status":        ww.Status(),
				"http_duration_ms":   duration.Milliseconds(),
				"http_bytes_written": ww.BytesWritten(),
			})

			// Определяем уровень логирования на основе статус кода
			statusCode := ww.Status()
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

					// Логируем панику как ошибку
					ctx = logger.WithFields(ctx, Fields{
						"panic":            rvr,
						"http_method":      r.Method,
						"http_url":         r.URL.String(),
						"http_path":        r.URL.Path,
						"http_remote_addr": r.RemoteAddr,
						"http_request_id":  middleware.GetReqID(ctx),
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

// shouldSkipPath проверяет, нужно ли пропустить путь при логировании
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
