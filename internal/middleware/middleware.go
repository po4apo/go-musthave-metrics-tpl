package middleware

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/po4apo/go-musthave-metrics-tpl/internal/utils/hasher"
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

			// Логирование body для методов с телом
			var bodyLog string
			if r.Body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
				const maxBodySize = 1 * 1024 * 1024 // 1MB
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
				if err == nil {
					bodyLog = string(bodyBytes)
					// Восстанавливаем body для следующего обработчика
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}

			logFields := []zap.Field{
				zap.String("method", method),
				zap.String("uri", uri),
			}
			if bodyLog != "" {
				logFields = append(logFields, zap.String("body", bodyLog))
			}

			logger.Info("Got request", logFields...)

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

type signingResponseWriter struct {
	http.ResponseWriter
	hasher      hasher.Hasher
	status      int
	wroteHeader bool
	body        bytes.Buffer
	maxBodySize int64
}

func (w *signingResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}
func (w *signingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	if int64(w.body.Len()+len(p)) > w.maxBodySize {
		return 0, errors.New("response body too large to sign")
	}
	return w.body.Write(p)
}

func CheckSign(h hasher.Hasher, logger zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1) Проверка подписи запроса (если подпись передана).
			// Поддерживаем оба заголовка: HashSHA256 (актуальный) и Hash (legacy в тестах).
			sig := strings.TrimSpace(r.Header.Get("HashSHA256"))
			if sig == "" {
				sig = strings.TrimSpace(r.Header.Get("Hash"))
			}
			if sig != "" && !strings.EqualFold(sig, "none") {
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					logger.Warn("Body read failed: %v", zap.Error(err), zap.String("method", r.Method), zap.String("uri", r.RequestURI))
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				ok, err := h.VerifyDataSignature(string(bodyBytes), sig)
				if err != nil || !ok {
					w.WriteHeader(http.StatusBadRequest)
					logger.Warn("Sign unverified", zap.Error(err), zap.String("method", r.Method), zap.String("uri", r.RequestURI))
					return
				}
			}
			// 2) Подпись ответа
			sw := &signingResponseWriter{
				ResponseWriter: w,
				hasher:         h,
				status:         http.StatusOK,
				wroteHeader:    false,
				maxBodySize:    1 << 20, // подними, если будет большой HTML на /
			}
			next.ServeHTTP(sw, r)
			// На случай, если хендлер вообще ничего не писал
			if !sw.wroteHeader {
				sw.status = http.StatusOK
				sw.wroteHeader = true
			}
			// Подпись от тела (несжатые байты)
			sign, err := sw.hasher.SignData(sw.body.Bytes())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logger.Warn("Data sign unseccessfully", zap.Error(err))
				return
			}
			// Ставим заголовок подписи ДО записи заголовков ответа
			sw.Header().Set("HashSHA256", sign)
			sw.ResponseWriter.WriteHeader(sw.status)
			_, _ = sw.ResponseWriter.Write(sw.body.Bytes())
		})
	}
}
