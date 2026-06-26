package core_http_middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if code == http.StatusNoContent || code == http.StatusNotModified {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.initGzip()
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.initGzip()
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) initGzip() {
	if w.writer == nil {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.Header().Add("Vary", "Accept-Encoding")
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter) // Перепривязываем к текущему ResponseWriter
		w.writer = gz
	}
}

func (w *gzipResponseWriter) Close() error {
	if w.writer != nil {
		err := w.writer.Close()
		gzipWriterPool.Put(w.writer) // Возвращаем writer обратно в пул
		return err
	}
	return nil
}

func Gzip() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}
			gzp := &gzipResponseWriter{ResponseWriter: w}
			defer gzp.Close()

			next.ServeHTTP(gzp, r)
		})
	}
}
