package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

// ListAssets godoc
// @Summary List assets
// @Tags Asset
// @Produce json
// @Param currency query string false "Currency" default(USD)
// @Param status query string false "Status active|inactive"
// @Success 200 {object} data.BaseResponse{data=[]data.AssetVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/assets [get]
func ListAssets(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	currency := c.Query("currency")
	items, err := service.GetAssetService().ListAssets(c.Request.Context(), currency, model.AssetStatusActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: items})
}

// GetAsset godoc
// @Summary Get asset detail
// @Tags Asset
// @Produce json
// @Param asset_id path int true "Asset ID"
// @Param currency query string false "Currency" default(USD)
// @Success 200 {object} data.BaseResponse{data=data.AssetVO}
// @Failure 401 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Router /asset-ms/v1/web/assets/{asset_id} [get]
func GetAsset(c *gin.Context) {
	if _, ok := requireUserID(c); !ok {
		return
	}
	assetID, err := strconv.ParseInt(c.Param("asset_id"), 10, 64)
	if err != nil || assetID <= 0 {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: "invalid asset_id"})
		return
	}
	item, err := service.GetAssetService().GetAsset(c.Request.Context(), assetID, c.Query("currency"))
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "mismatch") {
			status = http.StatusNotFound
		}
		c.JSON(status, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: item})
}
