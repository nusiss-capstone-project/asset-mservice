package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	_ "github.com/nusiss-capstone-project/asset-mservice/server/docs"
	"github.com/nusiss-capstone-project/asset-mservice/server/http/api"
	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	commonauth "github.com/nusiss-capstone-project/identity-mservice/common/auth"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	serviceURIPrefix = "/asset-ms/v1"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(log.RecoveryMiddleware())
	r.Use(otelgin.Middleware(data.ServiceName))
	r.Use(log.HTTPResponseIDMiddleware())
	r.Use(corsMiddleware())

	basicGroup := r.Group(serviceURIPrefix)
	{
		// High-frequency / non-business routes: no HTTP access log.
		basicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
		basicGroup.GET("/swagger/*any", gs.WrapHandler(
			swaggerFiles.Handler,
			gs.URL("/asset-ms/v1/swagger/doc.json"),
		))
	}

	// Business routes: enable request access logging.
	apiGroup := basicGroup.Group("")
	apiGroup.Use(log.HTTPObservabilityMiddleware())
	{
		webGroup := apiGroup.Group("/web")
		auth := commonauth.RequireUser()

		webGroup.GET("/assets", auth, api.ListAssets)
		webGroup.GET("/assets/:asset_id", auth, api.GetAsset)
		webGroup.POST("/quotes", auth, api.GenQuote)
		webGroup.POST("/orders", auth, api.CreateOrder)
		webGroup.GET("/orders", auth, api.ListOrders)
		webGroup.GET("/orders/:order_id", auth, api.GetOrder)
		webGroup.GET("/holdings", auth, api.ListHoldings)
		webGroup.POST("/deposit", auth, api.CreateDeposit)
		webGroup.GET("/fiat-accounts", auth, api.ListFiatAccounts)
		webGroup.GET("/ledgers", auth, api.ListLedgers)
	}
	return r
}

func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			commonauth.HeaderInternalUserID, commonauth.HeaderUserRole,
			log.RequestIDHeader, log.TraceIDHeader,
		},
		ExposeHeaders: []string{
			"Content-Length", commonauth.HeaderInternalUserID, commonauth.HeaderUserRole,
			log.RequestIDHeader, log.TraceIDHeader,
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func allowedOrigins() []string {
	if config.Config == nil || config.Config.SystemConfig == nil {
		return []string{}
	}
	return config.Config.SystemConfig.AllowedOrigins
}
