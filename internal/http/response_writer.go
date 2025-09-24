// Package http предоставляет внутренние HTTP утилиты
package http

import "net/http"

// ResponseWriter обертка для http.ResponseWriter для захвата статус кода и размера ответа
type ResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	bytesWritten  int
	headerWritten bool // флаг для предотвращения повторных вызовов WriteHeader
}

// NewResponseWriter создает новый wrapper для ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // По умолчанию 200
	}
}

// WriteHeader перехватывает статус код и предотвращает повторные вызовы
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	if rw.headerWritten {
		return // предотвращаем повторный вызов WriteHeader
	}
	rw.statusCode = statusCode
	rw.headerWritten = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write перехватывает количество записанных байт
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	// Если заголовки еще не были записаны, Write автоматически вызовет WriteHeader(200)
	if !rw.headerWritten {
		rw.headerWritten = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Status возвращает статус код ответа
func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}

// BytesWritten возвращает количество записанных байт
func (rw *ResponseWriter) BytesWritten() int {
	return rw.bytesWritten
}

// HeaderWritten возвращает true если заголовки уже были записаны
func (rw *ResponseWriter) HeaderWritten() bool {
	return rw.headerWritten
}
