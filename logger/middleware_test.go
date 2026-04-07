package logger

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHTTPMiddleware(t *testing.T) {
	// Создаем тестовый логгер
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})
	logrusLogger.SetLevel(logrus.DebugLevel)

	logger := &logrusAdapter{logger: logrusLogger}

	// Создаем middleware
	middleware := HTTPMiddleware(logger)

	// Создаем тестовый handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	// Выполняем запрос
	handler.ServeHTTP(rr, req)

	// Проверяем ответ
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Проверяем, что логирование произошло
	output := buf.String()
	if output == "" {
		t.Error("No log output generated")
	}

	// Проверяем наличие ожидаемых полей в логе
	expectedFields := []string{
		"http_method", "GET",
		"http_path", "/test",
		"http_status", "200",
		"http_user_agent", "test-agent",
		"HTTP request completed",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Missing field in log output: %s\nOutput: %s", field, output)
		}
	}
}

func TestHTTPMiddleware_Disabled(t *testing.T) {
	// Отключаем HTTP логирование
	os.Setenv("LOG_HTTP_ENABLED", "false")
	defer os.Unsetenv("LOG_HTTP_ENABLED")

	// Создаем тестовый логгер
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logger := &logrusAdapter{logger: logrusLogger}

	// Создаем middleware
	middleware := HTTPMiddleware(logger)

	// Создаем тестовый handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Выполняем запрос
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Проверяем, что логирование не произошло
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no log output when disabled, got: %s", output)
	}
}

func TestHTTPMiddleware_SkipPaths(t *testing.T) {
	// Создаем тестовый логгер
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logger := &logrusAdapter{logger: logrusLogger}

	// Создаем middleware
	middleware := HTTPMiddleware(logger)

	// Тестируем пропускаемые пути
	skipPaths := []string{"/metrics", "/health", "/healthz", "/ping"}

	for _, path := range skipPaths {
		t.Run("skip_"+path, func(t *testing.T) {
			buf.Reset() // Очищаем буфер для каждого теста

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Проверяем, что логирование не произошло
			output := buf.String()
			if output != "" {
				t.Errorf("Expected no log output for path %s, got: %s", path, output)
			}
		})
	}
}

func TestHTTPMiddleware_StatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
		expectedMsg   string
	}{
		{
			name:          "success 200",
			statusCode:    200,
			expectedLevel: "debug",
			expectedMsg:   "HTTP request completed",
		},
		{
			name:          "client error 404",
			statusCode:    404,
			expectedLevel: "warning",
			expectedMsg:   "HTTP request completed with client error",
		},
		{
			name:          "server error 500",
			statusCode:    500,
			expectedLevel: "error",
			expectedMsg:   "HTTP request completed with server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый логгер
			var buf bytes.Buffer
			logrusLogger := logrus.New()
			logrusLogger.SetOutput(&buf)
			logrusLogger.SetFormatter(&logrus.JSONFormatter{})
			logrusLogger.SetLevel(logrus.DebugLevel)
			logger := &logrusAdapter{logger: logrusLogger}

			// Создаем middleware
			middleware := HTTPMiddleware(logger)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			output := buf.String()

			// Проверяем уровень и сообщение
			if !strings.Contains(output, tt.expectedLevel) {
				t.Errorf("Expected log level %s not found in: %s", tt.expectedLevel, output)
			}
			if !strings.Contains(output, tt.expectedMsg) {
				t.Errorf("Expected message '%s' not found in: %s", tt.expectedMsg, output)
			}
			if !strings.Contains(output, fmt.Sprintf(`"http_status":%d`, tt.statusCode)) {
				t.Errorf("Expected status code %d not found in: %s", tt.statusCode, output)
			}
		})
	}
}

func TestRecovererMiddleware(t *testing.T) {
	// Создаем тестовый логгер
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})
	logger := &logrusAdapter{logger: logrusLogger}

	// Создаем middleware
	middleware := RecovererMiddleware(logger)

	// Создаем handler, который паникует
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	rr := httptest.NewRecorder()

	// Выполняем запрос (не должен паниковать)
	handler.ServeHTTP(rr, req)

	// Проверяем, что вернулся статус 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}

	// Проверяем, что паника была залогирована
	output := buf.String()
	if !strings.Contains(output, "test panic") {
		t.Errorf("Panic message not found in log: %s", output)
	}
	if !strings.Contains(output, "HTTP request panicked") {
		t.Errorf("Panic log message not found: %s", output)
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "RequestID key",
			key:      "RequestID",
			value:    "req-123",
			expected: "req-123",
		},
		{
			name:     "request-id key",
			key:      "request-id",
			value:    "req-456",
			expected: "req-456",
		},
		{
			name:     "X-Request-ID key",
			key:      "X-Request-ID",
			value:    "req-789",
			expected: "req-789",
		},
		{
			name:     "no request ID",
			key:      "",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.key != "" {
				ctx = context.WithValue(ctx, tt.key, tt.value)
			}

			result := getRequestID(ctx)
			if result != tt.expected {
				t.Errorf("getRequestID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/metrics", true},
		{"/health", true},
		{"/healthz", true},
		{"/ready", true},
		{"/readiness", true},
		{"/liveness", true},
		{"/ping", true},
		{"/metrics/detailed", true}, // prefix match
		{"/api/users", false},
		{"/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldSkipPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipPath(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
