package tracing

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// setupTracingTest настраивает трейсинг для тестов
func setupTracingTest(t *testing.T) func() {
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")

	return func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}
}

func TestStartSpan(t *testing.T) {
	// Тестируем создание span через helper функцию
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	if span == nil {
		t.Fatal("StartSpan should return a valid span")
	}

	spanContext := span.SpanContext()
	traceID := spanContext.TraceID().String()

	// КРИТИЧЕСКАЯ ПРОВЕРКА: trace ID НЕ должен быть нулевым
	if traceID == "00000000000000000000000000000000" {
		t.Fatal("StartSpan should generate valid trace ID, not all zeros")
	}

	if !spanContext.TraceID().IsValid() {
		t.Error("StartSpan should generate valid trace ID")
	}

	// Проверяем span ID
	spanID := spanContext.SpanID().String()
	if spanID == "0000000000000000" {
		t.Error("StartSpan should generate valid span ID, not all zeros")
	}

	if !spanContext.SpanID().IsValid() {
		t.Error("StartSpan should generate valid span ID")
	}
}

func TestStartSpan_Disabled(t *testing.T) {
	// Тестируем StartSpan с отключенным трейсингом
	os.Setenv("TRACING_ENABLED", "false")
	defer os.Unsetenv("TRACING_ENABLED")

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "disabled-operation")
	defer span.End()

	if span == nil {
		t.Fatal("StartSpan should return a span even when tracing is disabled")
	}

	// При отключенном трейсинге должен быть noop span
	spanContext := span.SpanContext()
	traceID := spanContext.TraceID().String()

	// У noop span trace ID будет невалидным (все нули)
	if traceID != "00000000000000000000000000000000" {
		t.Errorf("Disabled tracing should produce noop span with zero trace ID, got: %s", traceID)
	}
}

func TestWithSpan(t *testing.T) {
	// Тестируем WithSpan helper
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	var capturedSpan trace.Span
	var capturedTraceID string

	result := WithSpan(ctx, "with-span-test", func(ctx context.Context) error {
		capturedSpan = trace.SpanFromContext(ctx)
		capturedTraceID = capturedSpan.SpanContext().TraceID().String()

		// Проверяем, что span валиден внутри функции
		if capturedTraceID == "00000000000000000000000000000000" {
			t.Error("WithSpan should provide valid trace ID inside function")
		}

		return nil
	})

	if result != nil {
		t.Errorf("WithSpan should return nil error, got: %v", result)
	}

	if capturedSpan == nil {
		t.Fatal("WithSpan should provide valid span to function")
	}

	// Проверяем, что trace ID был валидным
	if capturedTraceID == "00000000000000000000000000000000" {
		t.Error("WithSpan should generate valid trace ID, not all zeros")
	}

	if !capturedSpan.SpanContext().TraceID().IsValid() {
		t.Error("WithSpan should generate valid trace ID")
	}
}

func TestWithSpan_Error(t *testing.T) {
	// Тестируем WithSpan с ошибкой
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	testError := errors.New("test error")
	var capturedTraceID string

	result := WithSpan(ctx, "with-span-error-test", func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		capturedTraceID = span.SpanContext().TraceID().String()

		// Даже при ошибке trace ID должен быть валидным
		if capturedTraceID == "00000000000000000000000000000000" {
			t.Error("WithSpan should provide valid trace ID even when function returns error")
		}

		return testError
	})

	if result != testError {
		t.Errorf("WithSpan should return function error, got: %v", result)
	}

	// Trace ID должен был быть валидным даже при ошибке
	if capturedTraceID == "00000000000000000000000000000000" {
		t.Error("WithSpan should generate valid trace ID even when function errors")
	}
}

func TestTraceFunction(t *testing.T) {
	// Тестируем TraceFunction helper
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	var capturedTraceID string

	result := TraceFunction(ctx, "TestFunction", func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		if span != nil {
			capturedTraceID = span.SpanContext().TraceID().String()
		}
		return nil
	})

	if result != nil {
		t.Errorf("TraceFunction should return nil error, got: %v", result)
	}

	// Проверяем, что trace ID был создан и валиден
	if capturedTraceID == "" {
		t.Error("TraceFunction should create span in context")
	}

	if capturedTraceID == "00000000000000000000000000000000" {
		t.Error("TraceFunction should generate valid trace ID, not all zeros")
	}
}

func TestTraceMethod(t *testing.T) {
	// Тестируем TraceMethod helper
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	var capturedTraceID string

	result := TraceMethod(ctx, "TestStruct.TestMethod", func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		if span != nil {
			capturedTraceID = span.SpanContext().TraceID().String()
		}
		return nil
	})

	if result != nil {
		t.Errorf("TraceMethod should return nil error, got: %v", result)
	}

	// Проверяем, что trace ID был создан и валиден
	if capturedTraceID == "" {
		t.Error("TraceMethod should create span in context")
	}

	if capturedTraceID == "00000000000000000000000000000000" {
		t.Error("TraceMethod should generate valid trace ID, not all zeros")
	}
}

func TestTraceDBQuery(t *testing.T) {
	// Тестируем TraceDBQuery helper
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	var capturedTraceID string

	result := TraceQuery(ctx, "SELECT * FROM users WHERE id = ?", func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)
		if span != nil {
			capturedTraceID = span.SpanContext().TraceID().String()
		}
		return nil
	})

	if result != nil {
		t.Errorf("TraceDBQuery should return nil error, got: %v", result)
	}

	// Проверяем, что trace ID был создан и валиден
	if capturedTraceID == "" {
		t.Error("TraceDBQuery should create span in context")
	}

	if capturedTraceID == "00000000000000000000000000000000" {
		t.Error("TraceDBQuery should generate valid trace ID, not all zeros")
	}
}

func TestHelpers_NestedSpans(t *testing.T) {
	// Тестируем вложенные span'ы через helper функции
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	var parentTraceID, childTraceID string

	// Создаем родительский span
	result := TraceFunction(ctx, "ParentFunction", func(parentCtx context.Context) error {
		parentSpan := trace.SpanFromContext(parentCtx)
		if parentSpan != nil {
			parentTraceID = parentSpan.SpanContext().TraceID().String()
		}

		// Создаем дочерний span
		return TraceMethod(parentCtx, "ChildStruct.ChildMethod", func(childCtx context.Context) error {
			childSpan := trace.SpanFromContext(childCtx)
			if childSpan != nil {
				childTraceID = childSpan.SpanContext().TraceID().String()
			}
			return nil
		})
	})

	if result != nil {
		t.Errorf("Nested spans should not return error, got: %v", result)
	}

	// Оба trace ID должны быть валидными
	if parentTraceID == "00000000000000000000000000000000" {
		t.Error("Parent span should have valid trace ID, not all zeros")
	}

	if childTraceID == "00000000000000000000000000000000" {
		t.Error("Child span should have valid trace ID, not all zeros")
	}

	// Дочерний span должен наследовать trace ID от родителя
	if parentTraceID != childTraceID {
		t.Errorf("Child span should inherit parent trace ID. Parent: %s, Child: %s",
			parentTraceID, childTraceID)
	}
}

func TestHelpers_Performance(t *testing.T) {
	// Тестируем производительность helper функций
	cleanup := setupTracingTest(t)
	defer cleanup()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()

	// Измеряем время выполнения множественных операций
	start := time.Now()

	for i := 0; i < 100; i++ {
		result := TraceFunction(ctx, "PerformanceTest", func(ctx context.Context) error {
			span := trace.SpanFromContext(ctx)
			traceID := span.SpanContext().TraceID().String()

			// Каждый trace ID должен быть валидным
			if traceID == "00000000000000000000000000000000" {
				t.Errorf("Iteration %d: trace ID should not be all zeros", i)
			}

			return nil
		})

		if result != nil {
			t.Errorf("Iteration %d: unexpected error: %v", i, result)
		}
	}

	duration := time.Since(start)

	// Проверяем, что операции выполняются достаточно быстро
	// 100 операций должны занимать меньше секунды
	if duration > time.Second {
		t.Errorf("100 tracing operations took too long: %v", duration)
	}
}

func TestHelpers_Disabled(t *testing.T) {
	// Тестируем helper функции с отключенным трейсингом
	os.Setenv("TRACING_ENABLED", "false")
	defer os.Unsetenv("TRACING_ENABLED")

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()

	// Все helper функции должны работать даже при отключенном трейсинге
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "StartSpan",
			fn: func() error {
				_, span := StartSpan(ctx, "disabled-start-span")
				defer span.End()
				return nil
			},
		},
		{
			name: "WithSpan",
			fn: func() error {
				return WithSpan(ctx, "disabled-with-span", func(ctx context.Context) error {
					return nil
				})
			},
		},
		{
			name: "TraceFunction",
			fn: func() error {
				return TraceFunction(ctx, "DisabledFunction", func(ctx context.Context) error {
					return nil
				})
			},
		},
		{
			name: "TraceMethod",
			fn: func() error {
				return TraceMethod(ctx, "DisabledStruct.DisabledMethod", func(ctx context.Context) error {
					return nil
				})
			},
		},
		{
			name: "TraceQuery",
			fn: func() error {
				return TraceQuery(ctx, "SELECT 1", func(ctx context.Context) error {
					return nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err != nil {
				t.Errorf("%s should work even when tracing is disabled, got error: %v", tt.name, err)
			}
		})
	}
}
