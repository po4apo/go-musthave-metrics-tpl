package middleware

import (
	"net/http"
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
