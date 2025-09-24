package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected default status code %d, got %d", http.StatusOK, rw.statusCode)
	}

	if rw.bytesWritten != 0 {
		t.Errorf("Expected bytes written to be 0, got %d", rw.bytesWritten)
	}

	if rw.headerWritten {
		t.Error("Expected headerWritten to be false initially")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	// Первый вызов WriteHeader
	rw.WriteHeader(http.StatusNotFound)

	if rw.Status() != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rw.Status())
	}

	if !rw.HeaderWritten() {
		t.Error("Expected headerWritten to be true after WriteHeader")
	}

	// Повторный вызов WriteHeader не должен изменить статус код
	rw.WriteHeader(http.StatusInternalServerError)

	if rw.Status() != http.StatusNotFound {
		t.Errorf("Expected status code to remain %d after second WriteHeader call, got %d",
			http.StatusNotFound, rw.Status())
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	data := []byte("Hello, World!")
	n, err := rw.Write(data)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.BytesWritten() != len(data) {
		t.Errorf("Expected BytesWritten to be %d, got %d", len(data), rw.BytesWritten())
	}

	if !rw.HeaderWritten() {
		t.Error("Expected headerWritten to be true after Write")
	}

	// Status должен остаться по умолчанию (200), так как WriteHeader не вызывался явно
	if rw.Status() != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rw.Status())
	}
}

func TestResponseWriter_WriteAfterWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	// Сначала устанавливаем статус код
	rw.WriteHeader(http.StatusCreated)

	// Затем пишем данные
	data := []byte("Created!")
	n, err := rw.Write(data)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.Status() != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rw.Status())
	}

	if rw.BytesWritten() != len(data) {
		t.Errorf("Expected BytesWritten to be %d, got %d", len(data), rw.BytesWritten())
	}
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	data1 := []byte("Hello, ")
	data2 := []byte("World!")

	n1, err1 := rw.Write(data1)
	if err1 != nil {
		t.Fatalf("Unexpected error on first write: %v", err1)
	}

	n2, err2 := rw.Write(data2)
	if err2 != nil {
		t.Fatalf("Unexpected error on second write: %v", err2)
	}

	expectedBytes := len(data1) + len(data2)
	if rw.BytesWritten() != expectedBytes {
		t.Errorf("Expected BytesWritten to be %d, got %d", expectedBytes, rw.BytesWritten())
	}

	if n1+n2 != expectedBytes {
		t.Errorf("Expected total written bytes %d, got %d", expectedBytes, n1+n2)
	}
}

func TestResponseWriter_PreventSuperfluousWriteHeader(t *testing.T) {
	// Тест для проверки того, что повторные вызовы WriteHeader не вызывают предупреждения
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	// Множественные вызовы WriteHeader не должны вызывать проблем
	rw.WriteHeader(http.StatusOK)
	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError)

	// Статус код должен остаться от первого вызова
	if rw.Status() != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rw.Status())
	}

	// После Write тоже не должно быть проблем
	data := []byte("test")
	rw.Write(data)
	rw.WriteHeader(http.StatusBadRequest) // это не должно иметь эффекта

	if rw.Status() != http.StatusOK {
		t.Errorf("Expected status code to remain %d, got %d", http.StatusOK, rw.Status())
	}
}
