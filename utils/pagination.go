package utils

import (
	"strconv"

	"github.com/Primeskills-Web-Team/golang-api-common/v2/constant"
	"github.com/gin-gonic/gin"
)

// SetPaginationHeader sets pagination-related headers in the HTTP response.
// It calculates the total number of pages, the next page, and the previous page
// based on the current page, limit, and total count of items.
//
// Headers set by this function:
// - X-Total-Count: Total number of items.
// - X-Total-Pages: Total number of pages.
// - X-Page: Current page number.
// - X-Limit: Number of items per page.
// - X-Next-Page: The next page number (if applicable).
// - X-Prev-Page: The previous page number (if applicable).
func SetPaginationHeader(ctx *gin.Context, page, limit, totalCount int) {
	var nextPage *int
	var prevPage *int
	totalPages := (totalCount + limit - 1) / limit

	if page < totalPages {
		next := page + 1
		nextPage = &next
	}

	if page > 1 {
		prev := page - 1
		prevPage = &prev
	}

	ctx.Header(constant.HeaderXTotalCount, strconv.Itoa(totalCount))
	ctx.Header(constant.HeaderXTotalPages, strconv.Itoa(totalPages))
	ctx.Header(constant.HeaderXPage, strconv.Itoa(page))
	ctx.Header(constant.HeaderXLimit, strconv.Itoa(limit))

	if nextPage != nil {
		ctx.Header(constant.HeaderXNextPage, strconv.Itoa(*nextPage))
	}

	if prevPage != nil {
		ctx.Header(constant.HeaderXPrevPage, strconv.Itoa(*prevPage))
	}
}
