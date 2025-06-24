package utils

import (
	"fmt"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/exception"
	"github.com/gin-gonic/gin"
)

// ExtractValidatedData extracts validated data from the gin context.
func ExtractValidatedData[V any](c *gin.Context) *V {
	v, ok := c.Get("validatedData")
	if !ok {
		return nil
	}

	val, ok := v.(*V)
	if !ok {
		return nil
	}

	return val
}

// GetPathParam retrieves a path parameter from the gin context and returns an error if it is missing.
func GetPathParam(c *gin.Context, paramName string) (string, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return "", exception.BadRequest(fmt.Sprintf("Missing required parameter: %s", paramName))
	}
	return paramValue, nil
}
