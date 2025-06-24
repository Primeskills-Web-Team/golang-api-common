package common

// ResponseDto : Struct for API response
type ResponseDto struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Errors  []ErrorValidationDto `json:"errors"`
	Data    interface{}          `json:"data"`
}

// ErrorValidationDto : Struct for error validation response
type ErrorValidationDto struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
