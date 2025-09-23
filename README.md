# lib-observability

Библиотека для observability Go сервисов, включающая логирование, метрики и трейсинг.

## Возможности

- 🌍 **Переменные окружения** - удобные функции для работы с environment variables
- 📝 **Логирование** - структурированное логирование с контекстом на основе logrus
- 📊 **Метрики** - интеграция с Prometheus для сбора метрик
- 🔍 **Трейсинг** - распределенная трассировка с OpenTelemetry

## Установка

```bash
go get github.com/Gen-Do/lib-obersvability
```

## Быстрый старт

### Самый простой способ (рекомендуется)

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/Gen-Do/lib-obersvability"
    "github.com/Gen-Do/lib-obersvability/env"
)

func main() {
    ctx := context.Background()
    
    // Инициализация всего observability stack
    obs := observability.MustNew(ctx)
    defer obs.Shutdown(ctx)
    
    // Создание роутера
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    
    // Полная настройка observability одной командой
    obs.SetupHTTP(r)
    
    // Ваши обработчики
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        obs.Logger().Info(r.Context(), "Hello World!")
        w.Write([]byte("Hello World!"))
    })
    
    port := env.Get("PORT", 8080)
    obs.Logger().Info(obs.Logger().WithField(ctx, "port", port), "Server starting")
    http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}
```

### Гибкий способ

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/Gen-Do/lib-obersvability"
    "github.com/Gen-Do/lib-obersvability/env"
)

func main() {
    ctx := context.Background()
    
    // Инициализация observability
    obs := observability.MustNew(ctx)
    defer obs.Shutdown(ctx)
    
    // Создание роутера и ручная настройка
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    
    // Добавляем middleware
    r.Use(obs.HTTPMiddleware())
    
    // Регистрируем служебные эндпоинты
    obs.RegisterRoutes(r)
    
    // Ваши обработчики
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        obs.Logger().Info(r.Context(), "Hello World!")
        w.Write([]byte("Hello World!"))
    })
    
    port := env.Get("PORT", 8080)
    obs.Logger().Info(ctx, "Server starting", "port", port)
    http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}
```

### Расширенный способ (ручная настройка)

```go
package main

import (
    "context"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/Gen-Do/lib-obersvability/env"
    "github.com/Gen-Do/lib-obersvability/logger"
    "github.com/Gen-Do/lib-obersvability/metrics"
    "github.com/Gen-Do/lib-obersvability/tracing"
)

func main() {
    ctx := context.Background()
    
    // Ручная инициализация компонентов
    log := logger.New(ctx)
    m := metrics.New()
    tracing.New()
    
    // Настройка роутера с middleware
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(logger.HTTPMiddleware(log))
    r.Use(m.Middleware())
    r.Use(tracing.HTTPMiddleware())
    
    // Эндпоинт для метрик
    r.Handle("/metrics", m.Handler())
    
    // Ваши обработчики
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        log.Info(r.Context(), "Hello World!")
        w.Write([]byte("Hello World!"))
    })
    
    port := env.Get("PORT", 8080)
    log.Info(ctx, "Server starting", "port", port)
    http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}
```

## Пакеты

### observability - Центральная структура (рекомендуется)

Единая точка входа для инициализации и управления всеми компонентами observability.

#### Использование

```go
// Создание и инициализация
obs, err := observability.New(ctx)
if err != nil {
    log.Fatal("Failed to initialize observability:", err)
}
defer obs.Shutdown(ctx)

// Или с паникой при ошибке
obs := observability.MustNew(ctx)
defer obs.Shutdown(ctx)

// Получение компонентов
logger := obs.GetLogger()  // или obs.Logger()
metrics := obs.GetMetrics() // или obs.Metrics()

// Проверка состояния трейсинга
if obs.IsTracingEnabled() {
    // трейсинг работает
}

// HTTP setup
r := chi.NewRouter()

// Способ 1: Полная автоматическая настройка (рекомендуется)
obs.SetupHTTP(r) // middleware + служебные эндпоинты

// Способ 2: Раздельная настройка
obs.RegisterRoutes(r) // только служебные эндпоинты
r.Use(obs.HTTPMiddleware()) // только middleware

// Способ 3: Единый middleware
r.Use(obs.HTTPMiddleware())

// Способ 4: Отдельные middleware (для тонкой настройки)
r.Use(obs.RecoveryMiddleware())
r.Use(obs.LoggingMiddleware())
r.Use(obs.MetricsMiddleware())
r.Use(obs.TracingMiddleware())

// Способ 5: Ручная регистрация эндпоинтов
r.Handle("/metrics", obs.MetricsHandler())  // Prometheus метрики
r.Handle("/health", obs.HealthHandler())    // Health check
r.Handle("/healthz", obs.HealthHandler())   // Kubernetes style
```

#### Преимущества централизованного подхода

- ✅ **Максимальная простота** - `obs.SetupHTTP(r)` настраивает всё одной командой
- ✅ **Автоматическое управление** - корректное завершение работы
- ✅ **Консистентность** - все компоненты работают согласованно
- ✅ **Удобство использования** - единый интерфейс для всех компонентов
- ✅ **Обработка ошибок** - централизованная обработка ошибок инициализации
- ✅ **Независимость от роутера** - работает с любым HTTP роутером через интерфейсы
- ✅ **Гибкость настройки** - от полной автоматизации до ручной настройки

#### Автоматические методы

- `obs.SetupHTTP(router)` - полная настройка (middleware + эндпоинты)
- `obs.RegisterRoutes(router)` - регистрация только служебных эндпоинтов
- `obs.HTTPMiddleware()` - единый middleware для всех компонентов

#### Служебные эндпоинты

Автоматически создаются при вызове `SetupHTTP()` или `RegisterRoutes()`:

- `GET /metrics` - Prometheus метрики
- `GET /health` - Health check endpoint
- `GET /healthz` - Health check (Kubernetes style)

Health endpoint возвращает:
```json
{
  "status": "ok",
  "service": "my-service",
  "version": "v1.0.0"
}
```

#### Совместимость с роутерами

Работает с любыми роутерами, поддерживающими интерфейсы:
- `RouterRegistrar` (методы Handle) - для регистрации эндпоинтов
- `HTTPRouter` (методы Handle + Use) - для полной настройки

Проверено с: chi, gorilla/mux, gin, fiber и другими.

### env - Переменные окружения

Удобные функции для работы с переменными окружения с поддержкой дефолтных значений.

#### Основные функции

```go
// Generic функция (рекомендуется)
port := env.Get("PORT", 8080)                    // int
host := env.Get("HOST", "localhost")             // string
enabled := env.Get("ENABLED", true)              // bool
timeout := env.Get("TIMEOUT_SEC", int64(30))     // int64
rate := env.Get("RATE", 0.5)                     // float64

// Специализированные функции (для обратной совместимости)
port := env.GetInt("PORT", 8080)
host := env.GetString("HOST", "localhost")
enabled := env.GetBool("ENABLED", true)
timeout := env.GetDuration("TIMEOUT", 30*time.Second)
```

#### Специальные функции

```go
serviceName := env.GetServiceName()    // SERVICE_NAME
version := env.GetServiceVersion()     // SERVICE_VERSION (default: "v0.0.1")
envName := env.GetEnvName()           // ENV_NAME (prod/staging/local, default: local)
```

### logger - Логирование

Структурированное логирование с поддержкой контекста на основе logrus.

#### Переменные окружения

- `LOG_LEVEL` - уровень логирования (default: "debug")
- `LOG_FORMAT` - формат логов: json/text (default: "json")
- `LOG_HTTP_ENABLED` - включить HTTP middleware (default: true)

#### Использование

```go
ctx := context.Background()
log := logger.New(ctx)

// Простое логирование
log.Info(ctx, "Simple message")
log.Error(ctx, "Error occurred")

// С дополнительными полями
ctx = log.WithField(ctx, "user_id", 123)
log.Info(ctx, "User action")

// С несколькими полями
ctx = log.WithFields(ctx, logger.Fields{
    "service": "auth",
    "version": "1.0.0",
})
log.Info(ctx, "Service started")

// С ошибкой
ctx = log.WithError(ctx, err)
log.Error(ctx, "Operation failed")
```

#### HTTP Middleware

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)              // Добавляет request ID (из chi)
r.Use(logger.HTTPMiddleware(log))        // Логирует HTTP запросы
r.Use(logger.RecovererMiddleware(log))   // Обрабатывает панику
```

### metrics - Метрики Prometheus

Интеграция с Prometheus для сбора метрик HTTP запросов и пользовательских метрик.

#### Переменные окружения

- `METRICS_ENABLED` - включить метрики (default: true)

#### Использование

```go
m := metrics.New()

// HTTP middleware (автоматически собирает метрики запросов)
r.Use(m.Middleware())

// Эндпоинт для метрик
r.Handle("/metrics", m.Handler())

// Добавление пользовательских метрик
registry := m.GetRegistry()
customCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "custom_operations_total",
        Help: "Total number of custom operations",
    },
    []string{"operation_type"},
)
registry.MustRegister(customCounter)

// Использование пользовательской метрики
customCounter.WithLabelValues("create").Inc()
```

#### Автоматические метрики

- `http_requests_total` - количество HTTP запросов
- `http_request_duration_seconds` - время выполнения запросов
- Стандартные метрики Go (память, горутины, GC)

### tracing - Трейсинг OpenTelemetry

Распределенная трассировка с OpenTelemetry.

#### Переменные окружения

- `TRACING_ENABLED` - включить трейсинг (default: false)
- `TRACING_SAMPLING_RATE` - частота сэмплирования (default: 1.0)
- `TRACING_ENDPOINT` - эндпоинт для отправки трейсов

#### Инициализация

```go
if err := tracing.New(); err != nil {
    log.Fatal(ctx, "Failed to initialize tracing", err)
}

// Корректное завершение
defer tracing.Shutdown(context.Background())
```

#### HTTP Middleware

```go
r.Use(tracing.HTTPMiddleware()) // Автоматически создает спаны для HTTP запросов
```

#### Ручное создание спанов

```go
// Базовое использование
ctx, span := tracing.StartSpan(ctx, "operation-name")
defer span.End()

// С помощью helper функций
err := tracing.WithSpan(ctx, "business-logic", func(ctx context.Context) error {
    // ваша бизнес-логика
    return nil
})

// Трейсинг функций
func MyFunction(ctx context.Context) error {
    return tracing.TraceFunction(ctx, "MyFunction", func(ctx context.Context) error {
        // логика функции
        return nil
    })
}

// Трейсинг методов
func (s *Service) MyMethod(ctx context.Context) error {
    return tracing.TraceMethod(ctx, "Service.MyMethod", func(ctx context.Context) error {
        // логика метода
        return nil
    })
}

// Трейсинг запросов к БД
err := tracing.TraceQuery(ctx, "SELECT * FROM users", func(ctx context.Context) error {
    // выполнение запроса
    return nil
})

// Трейсинг HTTP клиентских запросов
err := tracing.TraceHTTPClient(ctx, "GET", "https://api.example.com", func(ctx context.Context) error {
    // HTTP запрос
    return nil
})
```

#### Работа со спанами

```go
// Добавление атрибутов
tracing.AddSpanAttributes(ctx, 
    attribute.String("user.id", "123"),
    attribute.Int("batch.size", 100),
)

// Добавление событий
tracing.AddSpanEvent(ctx, "processing.started")

// Запись ошибок
tracing.RecordError(ctx, err)

// Получение trace ID для логирования
traceID := tracing.GetTraceID(ctx)
log.Info(log.WithField(ctx, "trace_id", traceID), "Processing completed")
```

## Переменные окружения

### Общие
- `SERVICE_NAME` - имя сервиса
- `SERVICE_VERSION` - версия сервиса (default: "v0.0.1")
- `ENV_NAME` - окружение: prod/staging/local (default: "local")

### Логирование
- `LOG_LEVEL` - debug/info/warn/error (default: "debug")
- `LOG_FORMAT` - json/text (default: "json")
- `LOG_HTTP_ENABLED` - true/false (default: true)

### Метрики
- `METRICS_ENABLED` - true/false (default: true)

### Трейсинг
- `TRACING_ENABLED` - true/false (default: false)
- `TRACING_SAMPLING_RATE` - 0.0-1.0 (default: 1.0)
- `TRACING_ENDPOINT` - URL эндпоинта для отправки трейсов

## Пример полной настройки

```bash
# .env файл
SERVICE_NAME=my-service
SERVICE_VERSION=v1.0.0
ENV_NAME=production

LOG_LEVEL=info
LOG_FORMAT=json
LOG_HTTP_ENABLED=true

METRICS_ENABLED=true

TRACING_ENABLED=true
TRACING_SAMPLING_RATE=0.1
TRACING_ENDPOINT=http://jaeger:14268/api/traces
```

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/Gen-Do/lib-obersvability/env"
    "github.com/Gen-Do/lib-obersvability/logger"
    "github.com/Gen-Do/lib-obersvability/metrics"
    "github.com/Gen-Do/lib-obersvability/tracing"
)

func main() {
    ctx := context.Background()
    
    // Инициализация
    log := logger.New(ctx)
    m := metrics.New()
    
    if err := tracing.New(); err != nil {
        log.Fatal(ctx, "Failed to initialize tracing", err)
    }
    defer tracing.Shutdown(ctx)
    
    // Настройка роутера
    r := chi.NewRouter()
    
    // Middleware (порядок важен!)
    r.Use(middleware.RequestID)
    r.Use(logger.RecovererMiddleware(log))
    r.Use(logger.HTTPMiddleware(log))
    r.Use(m.Middleware())
    r.Use(tracing.HTTPMiddleware())
    
    // Служебные эндпоинты
    r.Handle("/metrics", m.Handler())
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Бизнес-логика
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/users/{id}", getUserHandler(log))
        r.Post("/users", createUserHandler(log))
    })
    
    // Запуск сервера
    port := env.Get("PORT", 8080)
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", port),
        Handler: r,
    }
    
    // Graceful shutdown
    go func() {
        log.Info(ctx, "Server starting", "port", port)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(ctx, "Server failed", err)
        }
    }()
    
    // Ожидание сигнала завершения
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    
    log.Info(ctx, "Server shutting down")
    
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Error(ctx, "Server shutdown failed", err)
    }
    
    log.Info(ctx, "Server stopped")
}

func getUserHandler(log logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        userID := chi.URLParam(r, "id")
        
        err := tracing.TraceFunction(ctx, "getUserHandler", func(ctx context.Context) error {
            // Добавляем атрибуты в спан
            tracing.AddSpanAttributes(ctx, attribute.String("user.id", userID))
            
            // Логируем с контекстом
            ctx = log.WithField(ctx, "user_id", userID)
            log.Info(ctx, "Getting user")
            
            // Эмуляция работы
            time.Sleep(50 * time.Millisecond)
            
            return nil
        })
        
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(fmt.Sprintf(`{"id": "%s", "name": "User %s"}`, userID, userID)))
    }
}

func createUserHandler(log logger.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        err := tracing.TraceFunction(ctx, "createUserHandler", func(ctx context.Context) error {
            log.Info(ctx, "Creating user")
            
            // Эмуляция работы
            time.Sleep(100 * time.Millisecond)
            
            return nil
        })
        
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        w.Write([]byte(`{"id": "new-user", "name": "New User"}`))
    }
}
```

## Лицензия

MIT
