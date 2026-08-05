package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

// GenQuote godoc
// @Summary Generate quote
// @Tags Quote
// @Accept json
// @Produce json
// @Param body body data.GenQuoteRequest true "Quote request"
// @Success 200 {object} data.BaseResponse{data=data.QuoteVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/quotes [post]
func GenQuote(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req data.GenQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	quote, err := service.GetQuoteService().GenQuote(c.Request.Context(), userID, &req)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "redis") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: quote})
}
