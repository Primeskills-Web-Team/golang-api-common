package response

func ConstructResponse(code string, message string, errorDetail string, resource string, isSuccess bool, data interface{}) BaseResponse {
	return BaseResponse{
		Code:        code,
		IsSuccess:   false,
		Message:     message,
		ErrorDetail: errorDetail,
		Resource:    resource,
		Data:        data,
	}
}
