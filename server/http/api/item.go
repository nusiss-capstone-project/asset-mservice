package api

import (
	"net/http"
	"strconv"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/data"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/service"
	"github.com/gin-gonic/gin"
)

// Item Create.
//
// @Summary Item Create
// @Description Create an item record.
// @Tags Item
// @Accept json
// @Produce json
// @Param user body data.ItemVO true "Item information"
// @Success 200	{object} data.BaseResponse{data=data.ItemVO}
// @Failure 400 {object} data.BaseResponse{data=string}
// @Failure 500 {object} data.BaseResponse{data=string}
// @Router /__SERVICE_SLUG__/v1/items [post]
func CreateItem(c *gin.Context) {
	item := &data.ItemVO{}
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	ret, err := service.GetItemService().CreateItem(c, item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: map[string]interface{}{"item_id": ret}})
}

// Item Query with ID.
//
// @Summary Item Query with ID
// @Description Get item information by item ID.
// @Tags Item
// @Param item_id path int true "Item.ID"
// @Success 200
// @Router /__SERVICE_SLUG__/v1/items/{item_id} [get]
func GetItems(c *gin.Context) {
	itemId := c.Param("item_id")
	itemIdInt, err := strconv.Atoi(itemId)
	if err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: "Invalid item ID"})
		return
	}
	item, err := service.GetItemService().GetItemById(c.Request.Context(), itemIdInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: item})
}
