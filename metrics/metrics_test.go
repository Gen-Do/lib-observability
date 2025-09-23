package metrics

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	httputil "github.com/Gen-Do/lib-observability/internal/http"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNew(t *testing.T) {
	// Тест с включенными метриками
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")

	m := New()

	if m == nil {
		t.Error("New() returned nil")
	}

	if !m.enabled {
		t.Error("Expected metrics to be enabled")
	}

	if m.registry == nil {
		t.Error("Registry should not be nil when enabled")
	}
}

func TestNew_Disabled(t *testing.T) {
	// Тест с отключенными метриками
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	m := New()

	if m == nil {
		t.Error("New() returned nil")
	}

	if m.enabled {
		t.Error("Expected metrics to be disabled")
	}
}

func TestHandler(t *testing.T) {
	m := New()
	handler := m.Handler()

	if handler == nil {
		t.Error("Handler() returned nil")
	}

	// Тестируем HTTP запрос к handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Проверяем, что ответ содержит метрики Prometheus
	body := rr.Body.String()
	if !strings.Contains(body, "# HELP") {
		t.Error("Response should contain Prometheus metrics format")
	}
}

func TestHandler_Disabled(t *testing.T) {
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	m := New()
	handler := m.Handler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 when disabled, got %d", rr.Code)
	}
}

func TestMiddleware(t *testing.T) {
	m := New()

	// Создаем тестовый handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Применяем middleware
	middleware := m.Middleware()
	handler := middleware(testHandler)

	// Выполняем запрос
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Проверяем ответ
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", rr.Body.String())
	}

	// Даем время для инициализации метрик
	time.Sleep(10 * time.Millisecond)

	// Проверяем, что метрики были созданы и записаны
	registry := m.GetRegistry()
	if registry == nil {
		t.Error("Registry should not be nil")
		return
	}

	// Собираем метрики
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Errorf("Error gathering metrics: %v", err)
		return
	}

	// Ищем наши HTTP метрики
	var foundRequests, foundDuration bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "http_requests_total" {
			foundRequests = true
		}
		if mf.GetName() == "http_request_duration_seconds" {
			foundDuration = true
		}
	}

	if !foundRequests {
		t.Error("http_requests_total metric not found")
	}
	if !foundDuration {
		t.Error("http_request_duration_seconds metric not found")
	}
}

func TestMiddleware_Disabled(t *testing.T) {
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	m := New()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := m.Middleware()
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Handler должен работать нормально даже при отключенных метриках
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_SkipPaths(t *testing.T) {
	m := New()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware()
	handler := middleware(testHandler)

	skipPaths := []string{"/metrics", "/health", "/healthz", "/ping"}

	for _, path := range skipPaths {
		t.Run("skip_"+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Запрос должен пройти, но метрики не должны записываться
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestShouldSkipPath(t *testing.T) {
	m := New()

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
			result := m.shouldSkipPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipPath(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetRoutePath(t *testing.T) {
	m := New()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "normal path",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "empty path",
			path:     "/", // httptest.NewRequest не может создать запрос с пустым путем
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			result := m.getRoutePath(req)
			if result != tt.expected {
				t.Errorf("getRoutePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRegistry(t *testing.T) {
	m := New()
	registry := m.GetRegistry()

	if registry == nil {
		t.Error("GetRegistry() returned nil")
	}

	// Проверяем, что можем зарегистрировать пользовательскую метрику
	customCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_custom_counter",
		Help: "A test counter",
	})

	err := registry.Register(customCounter)
	if err != nil {
		t.Errorf("Failed to register custom metric: %v", err)
	}

	// Увеличиваем счетчик
	customCounter.Inc()

	// Проверяем, что метрика доступна
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Errorf("Error gathering metrics: %v", err)
		return
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_custom_counter" {
			found = true
			if len(mf.GetMetric()) > 0 {
				value := mf.GetMetric()[0].GetCounter().GetValue()
				if value != 1.0 {
					t.Errorf("Expected counter value 1.0, got %f", value)
				}
			}
			break
		}
	}

	if !found {
		t.Error("Custom metric not found in registry")
	}
}

func TestHTTPMetricsContent(t *testing.T) {
	// Пропускаем этот тест, так как метрики инициализируются глобально через sync.Once
	// и нельзя переинициализировать их с новыми переменными окружения
	t.Skip("Skipping test due to global metrics initialization via sync.Once")
}

func TestResponseWriter(t *testing.T) {
	// Тестируем наш собственный ResponseWriter
	rr := httptest.NewRecorder()
	rw := httputil.NewResponseWriter(rr)

	// Проверяем начальное состояние
	if rw.Status() != http.StatusOK {
		t.Errorf("Expected initial status 200, got %d", rw.Status())
	}

	if rw.BytesWritten() != 0 {
		t.Errorf("Expected initial bytes written 0, got %d", rw.BytesWritten())
	}

	// Пишем заголовок
	rw.WriteHeader(http.StatusCreated)
	if rw.Status() != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rw.Status())
	}

	// Пишем данные
	data := []byte("test response")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.BytesWritten() != len(data) {
		t.Errorf("Expected bytes written %d, got %d", len(data), rw.BytesWritten())
	}

	// Проверяем, что данные действительно записались
	if rr.Body.String() != "test response" {
		t.Errorf("Expected response body 'test response', got '%s'", rr.Body.String())
	}
}
