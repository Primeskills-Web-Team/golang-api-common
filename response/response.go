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
	"041": "Username or password is wrong",
	"047": "Invalid token or access",
	"043": "Forbidden Access resource",
	"042": "Please check your request body",
	"044": "Route not found",
	"046": "Data not found",
	"095": "Gateway timeout",
}

var HttpCode = map[string]int{
	"000": 200,
	"021": 201,
	"022": 200,
	"041": 401,
	"047": 401,
	"043": 403,
	"042": 422,
	"044": 404,
	"046": 404,
	"095": 504,
}
