package response

func ConstructResponse(code string, message string, errorDetail string, resource string, isSuccess bool, data interface{}) BaseResponse {
	return BaseResponse{
		Code:        code,
		IsSuccess:   isSuccess,
		Message:     message,
		ErrorDetail: errorDetail,
		Resource:    resource,
		Data:        data,
	}
}
