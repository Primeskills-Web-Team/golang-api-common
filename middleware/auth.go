package middleware

import (
	"fmt"
	"net/http"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/common"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/constant"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/httpclient"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/types"
	"github.com/Primeskills-Web-Team/golang-api-common/v2/utils"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware interface {
	Authorize(ctx *gin.Context)
}

type authMiddleware struct {
	httpClient     httpclient.HttpClient
	authServiceUrl string
}

// Authorize is a middleware function that checks the authorization token in the request header and send validation request to auth service.
func (a *authMiddleware) Authorize(ctx *gin.Context) {
	token := utils.GetBearerToken(ctx)
	if token == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, common.ResponseDto{
			Success: false,
			Message: "Authorization header is missing",
			Errors:  nil,
			Data:    nil,
		})
		return
	}

	// headers
	var mapHeaders = map[string]string{
		constant.HeaderAccept:        constant.ContentTypeJSON,
		constant.HeaderContentType:   constant.ContentTypeJSON,
		constant.HeaderAuthorization: fmt.Sprintf("Bearer %s", token),
	}

	// do request
	var user *types.User
	err := a.httpClient.Call(ctx.Request.Context(), http.MethodGet, a.authServiceUrl, mapHeaders, nil, &user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, common.ResponseDto{
			Success: false,
			Message: err.Error(),
			Errors:  nil,
			Data:    nil,
		})
		return
	}

	// set user to context
	ctx.Set("user", &user)
	ctx.Next()
}

func NewAuthMiddleware(
	httpClient httpclient.HttpClient,
	authServiceUrl string,
) AuthMiddleware {
	return &authMiddleware{
		httpClient: httpClient,
	}
}
