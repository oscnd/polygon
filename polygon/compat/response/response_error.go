package response

type ErrorResponse struct {
	Success *bool   `json:"success"`
	Message *string `json:"message,omitempty"`
	Error   *string `json:"error,omitempty"`
}
