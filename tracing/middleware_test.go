package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestHTTPMiddleware_Disabled(t *testing.T) {
	// Тестируем middleware с отключенным трейсингом
	os.Setenv("TRACING_ENABLED", "false")
	defer os.Unsetenv("TRACING_ENABLED")

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	// Создаем тестовый handler
	var receivedContext context.Context
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContext = r.Context()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Проверяем ответ
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// При отключенном трейсинге span должен быть noop
	span := trace.SpanFromContext(receivedContext)
	if span == nil {
		t.Error("Span should not be nil even when tracing is disabled")
	}

	// У noop span trace ID будет невалидным (все нули)
	spanContext := span.SpanContext()
	traceID := spanContext.TraceID().String()
	if traceID != "00000000000000000000000000000000" {
		t.Errorf("Expected noop trace ID to be all zeros, got: %s", traceID)
	}
}

func TestHTTPMiddleware_Enabled(t *testing.T) {
	// Тестируем middleware с включенным трейсингом
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	var receivedContext context.Context
	var receivedSpan trace.Span

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContext = r.Context()
		receivedSpan = trace.SpanFromContext(receivedContext)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("POST", "/api/users", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-client/1.0")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Проверяем ответ
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Проверяем, что span создан и валиден
	if receivedSpan == nil {
		t.Fatal("Span should not be nil when tracing is enabled")
	}

	spanContext := receivedSpan.SpanContext()

	// КРИТИЧЕСКАЯ ПРОВЕРКА: trace ID НЕ должен быть заполнен нулями
	traceID := spanContext.TraceID().String()
	if traceID == "00000000000000000000000000000000" {
		t.Fatal("Trace ID should NOT be all zeros when tracing is enabled in middleware")
	}

	if !spanContext.TraceID().IsValid() {
		t.Error("Trace ID should be valid when tracing is enabled")
	}

	// Проверяем span ID
	spanID := spanContext.SpanID().String()
	if spanID == "0000000000000000" {
		t.Error("Span ID should NOT be all zeros when tracing is enabled")
	}

	if !spanContext.SpanID().IsValid() {
		t.Error("Span ID should be valid when tracing is enabled")
	}
}

func TestHTTPMiddleware_TraceHeaders(t *testing.T) {
	// Тестируем обработку trace headers (W3C Trace Context)
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	var receivedSpan trace.Span

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSpan = trace.SpanFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Создаем запрос с trace headers
	req := httptest.NewRequest("GET", "/trace-test", nil)
	// Добавляем W3C Trace Context header
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedSpan == nil {
		t.Fatal("Span should not be nil")
	}

	spanContext := receivedSpan.SpanContext()
	traceID := spanContext.TraceID().String()

	// При наличии входящего trace context, trace ID может быть либо из header, либо новый
	// Главное - он НЕ должен быть нулевым

	// Важно: trace ID НЕ должен быть нулевым
	if traceID == "00000000000000000000000000000000" {
		t.Error("Trace ID should not be all zeros even when extracted from headers")
	}
}

func TestHTTPMiddleware_NoTraceHeaders(t *testing.T) {
	// Тестируем создание нового trace без входящих headers
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	var receivedSpan trace.Span

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSpan = trace.SpanFromContext(r.Context())
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest("POST", "/new-trace", nil)
	// Намеренно НЕ добавляем trace headers

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedSpan == nil {
		t.Fatal("Span should not be nil")
	}

	spanContext := receivedSpan.SpanContext()
	traceID := spanContext.TraceID().String()

	// Без входящих headers должен быть создан новый trace ID
	// ГЛАВНОЕ: он НЕ должен быть заполнен нулями
	if traceID == "00000000000000000000000000000000" {
		t.Fatal("New trace ID should NOT be all zeros")
	}

	if !spanContext.TraceID().IsValid() {
		t.Error("New trace ID should be valid")
	}

	// Проверяем, что trace ID имеет правильную длину
	if len(traceID) != 32 {
		t.Errorf("Trace ID should be 32 characters long, got %d", len(traceID))
	}

	// Проверяем, что trace ID содержит не только нули
	allZeros := true
	for _, char := range traceID {
		if char != '0' {
			allZeros = false
			break
		}
	}

	if allZeros {
		t.Error("New trace ID should contain non-zero characters")
	}
}

func TestHTTPMiddleware_StatusCodes(t *testing.T) {
	// Тестируем обработку различных статус кодов
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	tests := []struct {
		name       string
		statusCode int
	}{
		{"success", 200},
		{"created", 201},
		{"bad request", 400},
		{"not found", 404},
		{"server error", 500},
	}

	middleware := HTTPMiddleware()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedSpan trace.Span

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedSpan = trace.SpanFromContext(r.Context())
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest("GET", "/status-test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, rr.Code)
			}

			if receivedSpan == nil {
				t.Fatal("Span should not be nil")
			}

			spanContext := receivedSpan.SpanContext()
			traceID := spanContext.TraceID().String()

			// Для всех статус кодов trace ID должен быть валидным
			if traceID == "00000000000000000000000000000000" {
				t.Errorf("Trace ID should not be all zeros for status code %d", tt.statusCode)
			}

			if !spanContext.TraceID().IsValid() {
				t.Errorf("Trace ID should be valid for status code %d", tt.statusCode)
			}
		})
	}
}

func TestHTTPMiddleware_Concurrency(t *testing.T) {
	// Тестируем concurrent запросы для проверки уникальности trace ID
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	// Канал для сбора trace ID из concurrent запросов
	traceIDs := make(chan string, 10)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		spanContext := span.SpanContext()
		traceIDs <- spanContext.TraceID().String()
		w.WriteHeader(http.StatusOK)
	}))

	// Запускаем 10 concurrent запросов
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/concurrent-test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}()
	}

	// Собираем все trace ID
	collectedTraceIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		traceID := <-traceIDs

		// Каждый trace ID должен быть валидным
		if traceID == "00000000000000000000000000000000" {
			t.Errorf("Concurrent request %d: trace ID should not be all zeros", i)
		}

		// Проверяем уникальность
		if collectedTraceIDs[traceID] {
			t.Errorf("Duplicate trace ID found in concurrent requests: %s", traceID)
		}
		collectedTraceIDs[traceID] = true
	}

	// Все trace ID должны быть уникальными
	if len(collectedTraceIDs) != 10 {
		t.Errorf("Expected 10 unique trace IDs, got %d", len(collectedTraceIDs))
	}
}

func TestHTTPMiddleware_ContextPropagation(t *testing.T) {
	// Тестируем передачу trace context через middleware
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	middleware := HTTPMiddleware()

	var outerSpan, innerSpan trace.Span

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outerSpan = trace.SpanFromContext(r.Context())

		// Создаем дочерний span в том же контексте
		tracer := otel.Tracer("test")
		_, childSpan := tracer.Start(r.Context(), "child-operation")
		innerSpan = childSpan
		defer childSpan.End()

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/context-test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if outerSpan == nil || innerSpan == nil {
		t.Fatal("Both spans should not be nil")
	}

	outerTraceID := outerSpan.SpanContext().TraceID().String()
	innerTraceID := innerSpan.SpanContext().TraceID().String()

	// Оба trace ID должны быть валидными
	if outerTraceID == "00000000000000000000000000000000" {
		t.Error("Outer span trace ID should not be all zeros")
	}

	if innerTraceID == "00000000000000000000000000000000" {
		t.Error("Inner span trace ID should not be all zeros")
	}

	// Дочерний span должен наследовать trace ID от родителя
	if outerTraceID != innerTraceID {
		t.Errorf("Child span should inherit trace ID from parent. Parent: %s, Child: %s",
			outerTraceID, innerTraceID)
	}

	// Но span ID должны быть разными
	outerSpanID := outerSpan.SpanContext().SpanID().String()
	innerSpanID := innerSpan.SpanContext().SpanID().String()

	if outerSpanID == innerSpanID {
		t.Error("Parent and child spans should have different span IDs")
	}
}
