package utils

import (
	"strings"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/types"
	"github.com/gin-gonic/gin"
)

// GetBearerToken extracts the Bearer token from the Authorization header of the request context.
func GetBearerToken(ctx *gin.Context) string {
	token := ctx.Request.Header.Get("Authorization")
	if token == "" {
		return ""
	}

	return strings.TrimPrefix(token, "Bearer ")
}

// ExtractUserFromCtx extracts the user information from the context.
func ExtractUserFromCtx(ctx *gin.Context) *types.User {
	user, ok := ctx.Get("user")
	if !ok {
		return nil
	}

	val, ok := user.(*types.User)
	if !ok {
		return nil
	}

	return val
}
