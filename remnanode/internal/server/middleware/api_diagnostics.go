package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const diagnosticBodyLimit = 8192

var sensitiveJSONFieldPattern = regexp.MustCompile(`(?i)("(?:secret|secretKey|token|authorization|password|uuid|vlessUuid|prevVlessUuid|trojanPassword|ssPassword|id|key)"\s*:\s*")([^"]*)(")`)

// APIDiagnostics logs request/response details.
//
// debug=false: logs only failed API calls (status >= 400 or error bodies).
// debug=true:  logs every request/response pair at Info level.
//
// Request bodies are captured after GzipDecompress has already run, so they
// are normally plain JSON at this point. As a safety net, any body that still
// starts with the gzip magic bytes (0x1f 0x8b) is decompressed before logging.
func APIDiagnostics(debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		reqCapture := &limitedBuffer{limit: diagnosticBodyLimit}
		if c.Request.Body != nil {
			c.Request.Body = &captureReadCloser{
				ReadCloser: c.Request.Body,
				capture:    reqCapture,
			}
		}

		respCapture := &limitedBuffer{limit: diagnosticBodyLimit}
		originalWriter := c.Writer
		c.Writer = &captureResponseWriter{
			ResponseWriter: originalWriter,
			capture:        respCapture,
		}

		c.Next()

		status := c.Writer.Status()
		responseBody := respCapture.String()

		if !debug && !shouldLogAPIDiagnostic(status, responseBody) {
			return
		}

		// Decompress request body if it slipped through as raw gzip.
		reqBody := decompressIfGzip(reqCapture.String())

		var ev *zerolog.Event
		if status >= http.StatusBadRequest {
			ev = log.Warn()
		} else {
			ev = log.Info()
		}

		ev.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Int("status", status).
			Dur("latency", time.Since(start)).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Str("request_body", sanitizeDiagnosticBody(reqBody)).
			Bool("request_body_truncated", reqCapture.Truncated()).
			Str("response_body", sanitizeDiagnosticBody(responseBody)).
			Bool("response_body_truncated", respCapture.Truncated()).
			Msg("api request")
	}
}

// decompressIfGzip returns the gzip-decompressed string if data starts with the
// gzip magic bytes, otherwise returns the original string unchanged.
func decompressIfGzip(s string) string {
	if len(s) < 2 || s[0] != '\x1f' || s[1] != '\x8b' {
		return s
	}
	r, err := gzip.NewReader(strings.NewReader(s))
	if err != nil {
		return s
	}
	defer r.Close()
	decompressed, err := io.ReadAll(io.LimitReader(r, diagnosticBodyLimit))
	if err != nil {
		return s
	}
	return string(decompressed)
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	available := b.limit - b.buf.Len()
	if available > 0 {
		if len(p) <= available {
			_, _ = b.buf.Write(p)
		} else {
			_, _ = b.buf.Write(p[:available])
			b.truncated = true
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func (b *limitedBuffer) Truncated() bool {
	return b.truncated
}

type captureReadCloser struct {
	io.ReadCloser
	capture *limitedBuffer
}

func (r *captureReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		_, _ = r.capture.Write(p[:n])
	}
	return n, err
}

type captureResponseWriter struct {
	gin.ResponseWriter
	capture *limitedBuffer
}

func (w *captureResponseWriter) Write(data []byte) (int, error) {
	_, _ = w.capture.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *captureResponseWriter) WriteString(data string) (int, error) {
	_, _ = w.capture.Write([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func shouldLogAPIDiagnostic(status int, responseBody string) bool {
	if status >= http.StatusBadRequest {
		return true
	}

	body := strings.ToLower(responseBody)
	return strings.Contains(body, `"isstarted":false`) ||
		strings.Contains(body, `"success":false`) ||
		strings.Contains(body, `"isstopped":false`) ||
		strings.Contains(body, `"error":"`) ||
		strings.Contains(body, `"error": "`)
}

func sanitizeDiagnosticBody(body string) string {
	if body == "" {
		return ""
	}
	sanitized := sensitiveJSONFieldPattern.ReplaceAllString(body, `$1<redacted>$3`)
	sanitized = strings.ReplaceAll(sanitized, "\n", `\n`)
	sanitized = strings.ReplaceAll(sanitized, "\r", `\r`)
	return sanitized
}
