package tracing

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// WithSpan выполняет функцию внутри нового спана
// Удобно для обертывания бизнес-логики в спаны
func WithSpan(ctx context.Context, spanName string, fn func(ctx context.Context) error, opts ...trace.SpanStartOption) error {
	ctx, span := StartSpan(ctx, spanName, opts...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// AddSpanAttributes добавляет атрибуты к текущему спану в контексте
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent добавляет событие к текущему спану в контексте
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError записывает ошибку в текущий спан
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanStatus устанавливает статус текущего спана
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// GetTraceID возвращает trace ID из контекста (для логирования)
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID возвращает span ID из контекста (для логирования)
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// SpanFromContext возвращает текущий спан из контекста
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceFunction декоратор для автоматического трейсинга функций
// Использование:
//
//	func MyFunction(ctx context.Context) error {
//	    return tracing.TraceFunction(ctx, "MyFunction", func(ctx context.Context) error {
//	        // ваша логика здесь
//	        return nil
//	    })
//	}
func TraceFunction(ctx context.Context, functionName string, fn func(ctx context.Context) error) error {
	return WithSpan(ctx, functionName, fn, trace.WithSpanKind(trace.SpanKindInternal))
}

// TraceMethod декоратор для трейсинга методов структур
// Использование:
//
//	func (s *Service) MyMethod(ctx context.Context) error {
//	    return tracing.TraceMethod(ctx, "Service.MyMethod", func(ctx context.Context) error {
//	        // ваша логика здесь
//	        return nil
//	    })
//	}
func TraceMethod(ctx context.Context, methodName string, fn func(ctx context.Context) error) error {
	return WithSpan(ctx, methodName, fn, trace.WithSpanKind(trace.SpanKindInternal))
}

// TraceQuery декоратор для трейсинга запросов к базе данных
func TraceQuery(ctx context.Context, query string, fn func(ctx context.Context) error) error {
	return WithSpan(ctx, "db.query", func(ctx context.Context) error {
		AddSpanAttributes(ctx,
			attribute.String("db.statement", query),
			attribute.String("db.operation", "query"),
		)
		return fn(ctx)
	}, trace.WithSpanKind(trace.SpanKindClient))
}

// TraceHTTPClient декоратор для трейсинга HTTP клиентских запросов.
// fn получает контекст с активным span — используй InjectHTTPHeaders для propagation.
func TraceHTTPClient(ctx context.Context, method, url string, fn func(ctx context.Context) error) error {
	spanName := fmt.Sprintf("HTTP %s", method)
	return WithSpan(ctx, spanName, func(ctx context.Context) error {
		AddSpanAttributes(ctx,
			attribute.String("http.method", method),
			attribute.String("http.url", url),
			attribute.String("component", "http.client"),
		)
		return fn(ctx)
	}, trace.WithSpanKind(trace.SpanKindClient))
}

// InjectHTTPHeaders инжектирует текущий трейс-контекст (traceparent/tracestate)
// в заголовки исходящего HTTP запроса. Вызывай перед отправкой запроса в другой сервис.
//
// Пример:
//
//	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
//	tracing.InjectHTTPHeaders(ctx, req)
//	resp, err := client.Do(req)
func InjectHTTPHeaders(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// TracingTransport — http.RoundTripper который автоматически инжектирует
// трейс-заголовки в каждый исходящий запрос. Оберни им http.Client
// чтобы не вызывать InjectHTTPHeaders вручную.
//
// Пример:
//
//	client := &http.Client{
//	    Transport: tracing.NewTracingTransport(nil),
//	}
type TracingTransport struct {
	wrapped http.RoundTripper
}

// NewTracingTransport создаёт TracingTransport поверх переданного транспорта.
// Если wrapped == nil — используется http.DefaultTransport.
func NewTracingTransport(wrapped http.RoundTripper) *TracingTransport {
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return &TracingTransport{wrapped: wrapped}
}

// RoundTrip инжектирует traceparent/tracestate и выполняет запрос.
func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	return t.wrapped.RoundTrip(req)
}
