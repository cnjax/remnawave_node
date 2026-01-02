package errors

import "net/http"

// AppError represents a structured application error
type AppError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	HTTPCode int    `json:"-"`
}

func (e AppError) Error() string {
	return e.Message
}

// Predefined error codes matching TypeScript implementation
var (
	ErrInternalServer = AppError{
		Code:     "A001",
		Message:  "Server error",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrLoginError = AppError{
		Code:     "A002",
		Message:  "Login error",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrUnauthorized = AppError{
		Code:     "A003",
		Message:  "Unauthorized",
		HTTPCode: http.StatusUnauthorized,
	}
	ErrForbiddenRole = AppError{
		Code:     "A004",
		Message:  "Forbidden role error",
		HTTPCode: http.StatusForbidden,
	}
	ErrCreateAPIToken = AppError{
		Code:     "A005",
		Message:  "Create API token error",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrDeleteAPIToken = AppError{
		Code:     "A006",
		Message:  "Delete API token error",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrGetXrayStats = AppError{
		Code:     "A009",
		Message:  "Get Xray stats error",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetSystemStats = AppError{
		Code:     "A010",
		Message:  "Failed to get system stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetUsersStats = AppError{
		Code:     "A011",
		Message:  "Failed to get users stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetInboundStats = AppError{
		Code:     "A012",
		Message:  "Failed to get inbound stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetOutboundStats = AppError{
		Code:     "A013",
		Message:  "Failed to get outbound stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetInboundUsers = AppError{
		Code:     "A014",
		Message:  "Failed to get inbound users",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetInboundsStats = AppError{
		Code:     "A015",
		Message:  "Failed to get inbounds stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetOutboundsStats = AppError{
		Code:     "A016",
		Message:  "Failed to get outbounds stats",
		HTTPCode: http.StatusInternalServerError,
	}
	ErrFailedToGetCombinedStats = AppError{
		Code:     "A017",
		Message:  "Failed to get combined stats",
		HTTPCode: http.StatusInternalServerError,
	}
)
