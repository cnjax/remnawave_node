package errors

import (
	"github.com/gin-gonic/gin"
)

// CommandResponse represents the standard API response format
type CommandResponse struct {
	IsOk     bool        `json:"isOk"`
	Code     string      `json:"code,omitempty"`
	Message  string      `json:"message,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

// NewSuccessResponse creates a successful response
func NewSuccessResponse(response interface{}) CommandResponse {
	return CommandResponse{
		IsOk:     true,
		Response: response,
	}
}

// NewErrorResponse creates an error response from AppError
func NewErrorResponse(err AppError) CommandResponse {
	return CommandResponse{
		IsOk:    false,
		Code:    err.Code,
		Message: err.Message,
	}
}

// NewErrorResponseWithMessage creates an error response with custom message
func NewErrorResponseWithMessage(err AppError, message string) CommandResponse {
	return CommandResponse{
		IsOk:    false,
		Code:    err.Code,
		Message: message,
	}
}

// SendSuccess sends a successful JSON response
func SendSuccess(c *gin.Context, response interface{}) {
	c.JSON(200, NewSuccessResponse(response))
}

// SendError sends an error JSON response
func SendError(c *gin.Context, err AppError) {
	c.JSON(err.HTTPCode, NewErrorResponse(err))
}

// SendErrorWithMessage sends an error JSON response with custom message
func SendErrorWithMessage(c *gin.Context, err AppError, message string) {
	c.JSON(err.HTTPCode, NewErrorResponseWithMessage(err, message))
}
