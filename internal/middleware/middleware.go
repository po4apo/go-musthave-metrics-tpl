package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type (
	resposeData struct {
		size int
		code int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		rd *resposeData
	}
)

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.rd.size += size
	return size, err
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.rd.code = statusCode
}

func CustomLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lwr := loggingResponseWriter{
				ResponseWriter: w,
				rd:             &resposeData{0, 0},
			}
			start := time.Now()
			uri := r.RequestURI
			method := r.Method

			logger.Info(
				"Got request",
				zap.String("method", method),
				zap.String("uri", uri),
			)

			next.ServeHTTP(&lwr, r)

			delta := time.Since(start).String()

			logger.Info(
				"Send response",
				zap.Int("code", lwr.rd.code),
				zap.Int("size", lwr.rd.size),
				zap.String("duration", delta),
			)

		})
	}
}

// gzipResponseWriter оборачивает http.ResponseWriter и сжимает ответ gzip,
// если Content-Type ответа поддерживается (text/html, application/json).
type gzipResponseWriter struct {
	http.ResponseWriter
	gzWriter   *gzip.Writer
	compressed bool
}

func isSupportedContentType(ct string) bool {
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/json")
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	ct := w.Header().Get("Content-Type")
	if isSupportedContentType(ct) {
		w.compressed = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if w.compressed {
		return w.gzWriter.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

// Close закрывает gzip writer только если сжатие действительно применялось.
func (w *gzipResponseWriter) Close() error {
	if w.compressed {
		return w.gzWriter.Close()
	}
	return nil
}

// CompressGzip — middleware для gzip-сжатия ответов и декомпрессии входящих запросов.
func CompressGzip() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				gr, err := gzip.NewReader(r.Body)
				if err == nil {
					defer gr.Close()
					r.Body = gr
				}
			}

			// Сжатие ответа, если клиент поддерживает Accept-Encoding: gzip.
			ow := w
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gw := gzip.NewWriter(w)
				gzw := &gzipResponseWriter{
					ResponseWriter: w,
					gzWriter:       gw,
				}
				defer gzw.Close()
				ow = gzw
			}

			next.ServeHTTP(ow, r)
		})
	}
}
