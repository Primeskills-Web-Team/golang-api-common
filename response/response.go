package response

const (
	Success              = "000"
	SuccessCreate        = "021"
	SuccessUpdate        = "022"
	FailedAuthenticate   = "041"
	FailedAuthorization  = "047"
	DataNotFound         = "046"
	RouteNotFound        = "044"
	FailedValidationForm = "042"
	GeneralError         = "999"
	ForbiddenAccess      = "043"
	Timeout              = "094"
)

var Descriptions = map[string]string{
	"000": "Success",
	"021": "Success create data",
	"022": "Success update data",
	"041": "Invalid email or password",
	"047": "Invalid token or access",
	"043": "Forbidden Access resource",
	"042": "Please check your request body",
	"044": "Route not found",
	"046": "Data not found",
	"095": "Gateway timeout",
}

var HttpCode = map[string]int{
	Success:              200,
	SuccessCreate:        201,
	SuccessUpdate:        200,
	FailedAuthenticate:   401,
	FailedAuthorization:  401,
	ForbiddenAccess:      403,
	FailedValidationForm: 422,
	RouteNotFound:        404,
	DataNotFound:         404,
	Timeout:              504,
	GeneralError:         500,
}
