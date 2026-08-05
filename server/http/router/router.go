package router

import (
	"github.com/gin-gonic/gin"

	_ "github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/docs"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/api"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/data"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	serviceURIPrefix = "/__SERVICE_SLUG__/v1"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(log.RecoveryMiddleware())
	r.Use(otelgin.Middleware(data.ServiceName))
	r.Use(log.HTTPObservabilityMiddleware())
	r.Use(corsMiddleware())

	basicGroup := r.Group(serviceURIPrefix)
	{
		basicGroup.GET("/swagger/*any", gs.WrapHandler(
			swaggerFiles.Handler,
			gs.URL("/__SERVICE_SLUG__/v1/swagger/doc.json"),
		))
		basicGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
		basicGroup.POST("/items", api.CreateItem)
		basicGroup.GET("/items/:item_id", api.GetItems)
	}
	return r
}


func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins(),
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", log.RequestIDHeader,
		},
		ExposeHeaders: []string{
			"Content-Length", log.RequestIDHeader,
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