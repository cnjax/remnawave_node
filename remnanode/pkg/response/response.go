package response

// Response represents a standard API response
type Response struct {
	IsOk     bool        `json:"isOk"`
	Code     string      `json:"code,omitempty"`
	Message  string      `json:"message,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

// Success creates a successful response
func Success(data interface{}) Response {
	return Response{
		IsOk:     true,
		Response: data,
	}
}

// Error creates an error response
func Error(code, message string) Response {
	return Response{
		IsOk:    false,
		Code:    code,
		Message: message,
	}
}

// SuccessWithMessage creates a successful response with a message
func SuccessWithMessage(data interface{}, message string) Response {
	return Response{
		IsOk:     true,
		Message:  message,
		Response: data,
	}
}
