package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	kgzip "github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
)

// Decompress middleware decompresses gzip and zstd request bodies
func GzipDecompress() gin.HandlerFunc {
	// Create a reusable zstd decoder
	zstdDecoder, _ := zstd.NewReader(nil)

	return func(c *gin.Context) {
		contentEncoding := c.GetHeader("Content-Encoding")

		// Handle zstd encoding
		if contentEncoding == "zstd" {
			log.Debug().Msg("Decompressing zstd request body")

			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read request body for zstd")
				c.AbortWithStatusJSON(400, gin.H{"error": "Failed to read body"})
				return
			}

			decompressed, err := zstdDecoder.DecodeAll(bodyBytes, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to decompress zstd data")
				c.AbortWithStatusJSON(400, gin.H{"error": "Failed to decompress zstd data"})
				return
			}

			// Replace request body with decompressed data
			c.Request.Body = io.NopCloser(bytes.NewBuffer(decompressed))
			c.Request.ContentLength = int64(len(decompressed))
			c.Request.Header.Del("Content-Encoding")

			log.Debug().
				Int("compressed_size", len(bodyBytes)).
				Int("decompressed_size", len(decompressed)).
				Msg("Zstd decompression complete")

			c.Next()
			return
		}

		// Handle gzip encoding
		if contentEncoding == "gzip" {
			log.Debug().Msg("Decompressing gzip request body")

			gzReader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create gzip reader")
				c.AbortWithStatusJSON(400, gin.H{"error": "Invalid gzip data"})
				return
			}
			defer gzReader.Close()

			decompressed, err := io.ReadAll(gzReader)
			if err != nil {
				log.Error().Err(err).Msg("Failed to decompress gzip data")
				c.AbortWithStatusJSON(400, gin.H{"error": "Failed to decompress data"})
				return
			}

			c.Request.Body = io.NopCloser(bytes.NewBuffer(decompressed))
			c.Request.ContentLength = int64(len(decompressed))
			c.Request.Header.Del("Content-Encoding")

			log.Debug().Int("decompressed_size", len(decompressed)).Msg("Gzip decompression complete")

			c.Next()
			return
		}

		// Auto-detect compression by magic bytes if no Content-Encoding header
		if contentEncoding == "" && c.Request.Body != nil && c.Request.ContentLength > 4 {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
				c.Next()
				return
			}

			if len(bodyBytes) < 4 {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				c.Next()
				return
			}

			// Check for zstd magic number (0x28 0xb5 0x2f 0xfd)
			if bodyBytes[0] == 0x28 && bodyBytes[1] == 0xb5 && bodyBytes[2] == 0x2f && bodyBytes[3] == 0xfd {
				log.Debug().Msg("Auto-detected zstd compression")

				decompressed, err := zstdDecoder.DecodeAll(bodyBytes, nil)
				if err != nil {
					log.Error().Err(err).Msg("Failed to decompress auto-detected zstd")
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					c.Next()
					return
				}

				c.Request.Body = io.NopCloser(bytes.NewBuffer(decompressed))
				c.Request.ContentLength = int64(len(decompressed))
				log.Debug().Int("decompressed_size", len(decompressed)).Msg("Auto-detected zstd decompression complete")
				c.Next()
				return
			}

			// Check for gzip magic number (0x1f 0x8b)
			if bodyBytes[0] == 0x1f && bodyBytes[1] == 0x8b {
				log.Debug().Msg("Auto-detected gzip compression")

				gzReader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
				if err != nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					c.Next()
					return
				}
				defer gzReader.Close()

				decompressed, err := io.ReadAll(gzReader)
				if err != nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					c.Next()
					return
				}

				c.Request.Body = io.NopCloser(bytes.NewBuffer(decompressed))
				c.Request.ContentLength = int64(len(decompressed))
				log.Debug().Int("decompressed_size", len(decompressed)).Msg("Auto-detected gzip decompression complete")
				c.Next()
				return
			}

			// Not compressed, restore original body
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()
	}
}

// gzipWriter wraps gin.ResponseWriter and compresses the response body.
type gzipWriter struct {
	gin.ResponseWriter
	gz *kgzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.gz.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.gz.Write([]byte(s))
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		gz, _ := kgzip.NewWriterLevel(nil, kgzip.DefaultCompression)
		return gz
	},
}

// GzipCompress adds gzip response compression when the client sends
// Accept-Encoding: gzip, matching TS app.use(compression()) behavior.
func GzipCompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		gz := gzipPool.Get().(*kgzip.Writer)
		gz.Reset(c.Writer)

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		// Content-Length is unknown after compression; remove to avoid mismatch.
		c.Writer.Header().Del("Content-Length")

		gw := &gzipWriter{ResponseWriter: c.Writer, gz: gz}
		c.Writer = gw

		defer func() {
			// Ensure the gzip footer is written even if handler panics.
			gz.Close()
			gzipPool.Put(gz)
		}()

		c.Next()
	}
}
