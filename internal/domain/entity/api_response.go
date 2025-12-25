package entity

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewSuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(code string, message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Message: message,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}
