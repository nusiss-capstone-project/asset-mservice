package http

import (
	"fmt"
	"os"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/config"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/router"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
)

func Init(exitSig chan os.Signal) {
	r := router.NewRouter()
	log.Logger.Infof("Cerami Craft ItemService start...")
	err := r.Run(fmt.Sprintf("%s:%d", config.Config.HttpConfig.Host, config.Config.HttpConfig.Port))
	if err != nil {
		log.Logger.Fatalf("Failed to run server: %v", err)
		exitSig <- os.Interrupt
	}
}
