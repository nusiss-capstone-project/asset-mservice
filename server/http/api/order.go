package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

// CreateOrder godoc
// @Summary Create order
// @Tags Order
// @Accept json
// @Produce json
// @Param body body data.CreateOrderRequest true "Order request"
// @Success 200 {object} data.BaseResponse{data=data.OrderVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/orders [post]
func CreateOrder(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req data.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	order, err := service.GetOrderService().CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: order})
}

// GetOrder godoc
// @Summary Get order detail
// @Tags Order
// @Produce json
// @Param order_id path int true "Order ID"
// @Success 200 {object} data.BaseResponse{data=data.OrderDetailVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/orders/{order_id} [get]
func GetOrder(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: "invalid order_id"})
		return
	}
	order, err := service.GetOrderService().GetOrderDetail(c.Request.Context(), userID, orderID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: order})
}

// ListOrders godoc
// @Summary List orders
// @Tags Order
// @Produce json
// @Param status query string false "pending|pay_succeed|pay_fail"
// @Param asset_id query int false "Asset ID"
// @Param created_from query int false "Unix timestamp"
// @Param created_to query int false "Unix timestamp"
// @Param cursor query string false "Last seen order_id"
// @Param limit query int false "Page size" default(20)
// @Success 200 {object} data.BaseResponse{data=data.OrderListVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/orders [get]
func ListOrders(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	assetID, _ := strconv.ParseInt(c.Query("asset_id"), 10, 64)
	createdFrom, _ := strconv.ParseInt(c.Query("created_from"), 10, 64)
	createdTo, _ := strconv.ParseInt(c.Query("created_to"), 10, 64)
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	list, err := service.GetOrderService().ListOrders(c.Request.Context(), service.ListOrdersQuery{
		UserID:      userID,
		Status:      c.Query("status"),
		AssetID:     assetID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		Cursor:      cursor,
		Limit:       limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: list})
}
