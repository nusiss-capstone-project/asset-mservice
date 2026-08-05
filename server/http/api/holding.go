package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

// ListHoldings godoc
// @Summary List holdings
// @Tags Holding
// @Produce json
// @Param asset_id query int false "Asset ID"
// @Success 200 {object} data.BaseResponse{data=data.HoldingListVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/holdings [get]
func ListHoldings(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	assetID, _ := strconv.ParseInt(c.Query("asset_id"), 10, 64)
	list, err := service.GetHoldingService().ListHoldings(c.Request.Context(), userID, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: list})
}
