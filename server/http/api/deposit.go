package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

// CreateDeposit godoc
// @Summary Create fiat deposit
// @Tags Deposit
// @Accept json
// @Produce json
// @Param body body data.CreateDepositRequest true "Deposit request"
// @Success 200 {object} data.BaseResponse{data=data.FiatTransactionVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/deposit [post]
func CreateDeposit(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req data.CreateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	txn, err := service.GetDepositService().CreateDeposit(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: txn})
}

// ListFiatAccounts godoc
// @Summary List fiat accounts for current market
// @Tags Deposit
// @Produce json
// @Param currencies query string false "Comma-separated currencies"
// @Success 200 {object} data.BaseResponse{data=data.FiatAccountListVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/fiat-accounts [get]
func ListFiatAccounts(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	list, err := service.GetDepositService().ListFiatAccounts(c.Request.Context(), userID, c.Query("currencies"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: list})
}

// ListLedgers godoc
// @Summary List account ledger entries
// @Tags Deposit
// @Produce json
// @Param cursor query string false "Last seen ledger_id"
// @Param limit query int false "Page size" default(20)
// @Param asset_code query string false "USD or BTC"
// @Param business_type query string false "DEPOSIT|PURCHASE|REWARD"
// @Param from query int false "Unix timestamp"
// @Param to query int false "Unix timestamp"
// @Success 200 {object} data.BaseResponse{data=data.LedgerListVO}
// @Failure 401 {object} data.BaseResponse
// @Router /asset-ms/v1/web/ledgers [get]
func ListLedgers(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	list, err := service.GetDepositService().ListLedgers(c.Request.Context(), service.ListLedgersQuery{
		UserID:       userID,
		AssetCode:    strings.TrimSpace(c.Query("asset_code")),
		BusinessType: strings.TrimSpace(c.Query("business_type")),
		CreatedFrom:  from,
		CreatedTo:    to,
		Cursor:       cursor,
		Limit:        limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{Code: -1, ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: list})
}
