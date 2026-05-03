package errors

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// errorPayload matches TS http-exception.filter.ts shape:
// { timestamp, path, message, errorCode }
type errorPayload struct {
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

func newErrorPayload(c *gin.Context, message, errorCode string) errorPayload {
	return errorPayload{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Path:      c.Request.URL.Path,
		Message:   message,
		ErrorCode: errorCode,
	}
}

// SendSuccess sends a successful JSON response matching TS shape: {response: data}
func SendSuccess(c *gin.Context, response interface{}) {
	c.JSON(http.StatusOK, gin.H{"response": response})
}

// SendError sends an error JSON response
func SendError(c *gin.Context, err AppError) {
	c.JSON(err.HTTPCode, newErrorPayload(c, err.Message, err.Code))
}

// SendErrorWithMessage sends an error JSON response with custom message
func SendErrorWithMessage(c *gin.Context, err AppError, message string) {
	c.JSON(err.HTTPCode, newErrorPayload(c, message, err.Code))
}
