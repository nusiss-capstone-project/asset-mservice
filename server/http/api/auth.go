package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	commonauth "github.com/nusiss-capstone-project/identity-mservice/common/auth"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
)

func requireUserID(c *gin.Context) (int64, bool) {
	userID, ok := commonauth.GetUserID(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, data.BaseResponse{Code: -1, ErrMsg: "unauthorized"})
		return 0, false
	}
	return userID, true
}
