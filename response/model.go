package response

type BaseResponse struct {
	Ref         string      `json:"ref"`
	Code        string      `json:"code"`
	IsSuccess   bool        `json:"is_success"`
	Message     string      `json:"message"`
	ErrorDetail string      `json:"error_detail"`
	Resource    string      `json:"resource"`
	Data        interface{} `json:"data"`
}
