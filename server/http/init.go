package http

import (
	"fmt"
	"os"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/http/router"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
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
