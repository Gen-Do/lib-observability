package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// testLogger создает логгер для тестирования с буфером для захвата вывода
func testLogger() (Logger, *bytes.Buffer) {
	var buf bytes.Buffer

	// Создаем logrus логгер с буфером
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})
	logrusLogger.SetLevel(logrus.DebugLevel)

	// Создаем адаптер
	adapter := &logrusAdapter{logger: logrusLogger}

	return adapter, &buf
}

func TestNew(t *testing.T) {
	ctx := context.Background()

	// Устанавливаем переменные окружения для теста
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("SERVICE_VERSION", "v1.0.0")
	os.Setenv("ENV_NAME", "test")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_VERSION")
		os.Unsetenv("ENV_NAME")
	}()

	logger := New(ctx)

	if logger == nil {
		t.Error("New() returned nil logger")
	}
}

func TestWithField(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Добавляем поле в контекст
	ctx = logger.WithField(ctx, "user_id", 123)

	// Логируем с этим контекстом
	logger.Info(ctx, "Test message")

	// Проверяем, что поле попало в лог
	output := buf.String()
	if !strings.Contains(output, "user_id") {
		t.Errorf("WithField() did not add field to context, output: %s", output)
	}
	if !strings.Contains(output, "123") {
		t.Errorf("WithField() did not add field value to context, output: %s", output)
	}
	if !strings.Contains(output, "Test message") {
		t.Errorf("Message not found in output: %s", output)
	}
}

func TestWithFields(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Добавляем несколько полей в контекст
	fields := Fields{
		"user_id": 123,
		"action":  "login",
		"ip":      "192.168.1.1",
	}
	ctx = logger.WithFields(ctx, fields)

	// Логируем с этим контекстом
	logger.Info(ctx, "User action performed")

	// Проверяем, что все поля попали в лог
	output := buf.String()
	expectedFields := []string{"user_id", "123", "action", "login", "ip", "192.168.1.1"}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("WithFields() missing field %s in output: %s", field, output)
		}
	}
}

func TestWithError(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	testError := errors.New("test error message")

	// Добавляем ошибку в контекст
	ctx = logger.WithError(ctx, testError)

	// Логируем с этим контекстом
	logger.Error(ctx, "Operation failed")

	// Проверяем, что ошибка попала в лог
	output := buf.String()
	if !strings.Contains(output, "test error message") {
		t.Errorf("WithError() did not add error to context, output: %s", output)
	}
	if !strings.Contains(output, "Operation failed") {
		t.Errorf("Message not found in output: %s", output)
	}
}

func TestContextChaining(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Тестируем цепочку вызовов WithField -> WithFields -> WithError
	ctx = logger.WithField(ctx, "step", 1)
	ctx = logger.WithFields(ctx, Fields{"user_id": 456, "session": "abc123"})
	ctx = logger.WithError(ctx, errors.New("chain error"))

	// Логируем с накопленным контекстом
	logger.Warn(ctx, "Complex operation")

	// Проверяем, что все данные сохранились
	output := buf.String()
	expectedData := []string{
		"step", "1",
		"user_id", "456",
		"session", "abc123",
		"chain error",
		"Complex operation",
	}

	for _, data := range expectedData {
		if !strings.Contains(output, data) {
			t.Errorf("Context chaining missing data %s in output: %s", data, output)
		}
	}
}

func TestContextOverwrite(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Добавляем поле
	ctx = logger.WithField(ctx, "user_id", 123)

	// Перезаписываем то же поле через WithFields
	ctx = logger.WithFields(ctx, Fields{"user_id": 456})

	// Логируем
	logger.Info(ctx, "Overwrite test")

	// Проверяем, что значение перезаписалось
	output := buf.String()
	if !strings.Contains(output, "456") {
		t.Errorf("Field overwrite failed, output: %s", output)
	}
	// Старое значение не должно присутствовать (в зависимости от реализации)
	// Это может варьироваться в зависимости от того, как работает logrus
}

func TestMultipleLogCallsWithSameContext(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Добавляем поля в контекст
	ctx = logger.WithField(ctx, "request_id", "req-123")
	ctx = logger.WithField(ctx, "user_id", 789)

	// Делаем несколько логирований с тем же контекстом
	logger.Debug(ctx, "Debug message")
	logger.Info(ctx, "Info message")
	logger.Warn(ctx, "Warning message")

	// Проверяем, что все сообщения содержат поля из контекста
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		if !strings.Contains(line, "request_id") || !strings.Contains(line, "req-123") {
			t.Errorf("Line %d missing request_id: %s", i, line)
		}
		if !strings.Contains(line, "user_id") || !strings.Contains(line, "789") {
			t.Errorf("Line %d missing user_id: %s", i, line)
		}
	}
}

func TestLogLevels(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	ctx = logger.WithField(ctx, "level_test", true)

	// Тестируем все уровни логирования
	logger.Debug(ctx, "Debug level")
	logger.Info(ctx, "Info level")
	logger.Print(ctx, "Print level") // должен работать как Info
	logger.Warn(ctx, "Warn level")
	logger.Error(ctx, "Error level")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Должно быть 5 строк (Print = Info, поэтому отдельной строки не будет в некоторых реализациях)
	expectedMessages := []string{"Debug level", "Info level", "Print level", "Warn level", "Error level"}

	for _, msg := range expectedMessages {
		if !strings.Contains(output, msg) {
			t.Errorf("Missing log message: %s", msg)
		}
	}

	// Проверяем, что в каждой строке есть поле из контекста
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "level_test") {
			t.Errorf("Line missing context field: %s", line)
		}
	}
}

func TestEmptyContext(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	// Логируем без добавления полей в контекст
	logger.Info(ctx, "Empty context message")

	output := buf.String()
	if !strings.Contains(output, "Empty context message") {
		t.Errorf("Message not found with empty context: %s", output)
	}
}

func TestComplexScenario(t *testing.T) {
	// Тестируем сценарий, похожий на реальное использование
	logger, buf := testLogger()
	ctx := context.Background()

	// Имитируем HTTP запрос
	ctx = logger.WithFields(ctx, Fields{
		"method": "POST",
		"path":   "/api/users",
		"ip":     "10.0.0.1",
	})

	logger.Info(ctx, "Request started")

	// Имитируем обработку с дополнительными данными
	ctx = logger.WithField(ctx, "user_id", 12345)
	logger.Debug(ctx, "User authenticated")

	// Имитируем ошибку
	ctx = logger.WithError(ctx, errors.New("database connection failed"))
	logger.Error(ctx, "Failed to create user")

	// Проверяем финальный результат
	output := buf.String()

	// Должны быть все данные во всех сообщениях
	requiredData := []string{
		"method", "POST",
		"path", "/api/users",
		"ip", "10.0.0.1",
		"Request started",
		"user_id", "12345",
		"User authenticated",
		"database connection failed",
		"Failed to create user",
	}

	for _, data := range requiredData {
		if !strings.Contains(output, data) {
			t.Errorf("Complex scenario missing data: %s\nFull output: %s", data, output)
		}
	}
}

func TestJSONStructure(t *testing.T) {
	logger, buf := testLogger()
	ctx := context.Background()

	ctx = logger.WithFields(ctx, Fields{
		"string_field": "test_value",
		"int_field":    42,
		"bool_field":   true,
	})

	logger.Info(ctx, "JSON structure test")

	// Проверяем, что вывод является валидным JSON
	output := strings.TrimSpace(buf.String())

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v\nOutput: %s", err, output)
		return
	}

	// Проверяем наличие ожидаемых полей
	if logEntry["string_field"] != "test_value" {
		t.Errorf("string_field mismatch: got %v, want test_value", logEntry["string_field"])
	}

	if logEntry["int_field"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("int_field mismatch: got %v, want 42", logEntry["int_field"])
	}

	if logEntry["bool_field"] != true {
		t.Errorf("bool_field mismatch: got %v, want true", logEntry["bool_field"])
	}

	if logEntry["msg"] != "JSON structure test" {
		t.Errorf("msg mismatch: got %v, want 'JSON structure test'", logEntry["msg"])
	}
}
