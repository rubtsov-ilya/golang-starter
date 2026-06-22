package core_http_response

import (
	"bytes"
	"net/http"
	"sync"
)

// TimeoutResponseWriter — обёртка над http.ResponseWriter для поддержки таймаутов запросов.
// Накапливает заголовки и тело в буфере в памяти. Если запрос выполняется вовремя —
// сбрасывает (flush) всё в сеть. Если срабатывает таймаут — игнорирует последующую запись
// обработчика, позволяя вернуть клиенту 504.
type TimeoutResponseWriter struct {
	http.ResponseWriter

	mu         sync.Mutex
	buf        bytes.Buffer
	statusCode int
	timedOut   bool
}

// NewTimeoutResponseWriter создаёт новый TimeoutResponseWriter.
func NewTimeoutResponseWriter(w http.ResponseWriter) *TimeoutResponseWriter {
	return &TimeoutResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader сохраняет статус-код в памяти, не отправляя его в сеть сразу.
func (tw *TimeoutResponseWriter) WriteHeader(statusCode int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return
	}
	tw.statusCode = statusCode
}

// Write записывает тело ответа в буфер памяти.
func (tw *TimeoutResponseWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	return tw.buf.Write(b)
}

// TryTimeout пытается перевести райтер в состояние таймаута.
// Возвращает true, если мы успели первыми заблокировать запись.
func (tw *TimeoutResponseWriter) TryTimeout() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return false
	}
	tw.timedOut = true
	return true
}

// FlushResponse отправляет накопленные заголовки и тело в оригинальный ResponseWriter.
func (tw *TimeoutResponseWriter) FlushResponse() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return http.ErrHandlerTimeout
	}

	tw.ResponseWriter.WriteHeader(tw.statusCode)
	_, err := tw.ResponseWriter.Write(tw.buf.Bytes())
	return err
}
