package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/grpc"
	"github.com/nusiss-capstone-project/asset-mservice/server/http"
	"github.com/nusiss-capstone-project/asset-mservice/server/kafka/listener"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/nusiss-capstone-project/asset-mservice/server/telemetry"
)

var (
	sigCh = make(chan os.Signal, 1)
)

func main() {
	config.Init()
	log.InitLogger()
	repository.Init()
	cacheredis.Init()

	shutdownTelemetry := telemetry.Init(context.Background())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			log.Logger.Errorw("telemetry shutdown failed", "error", err)
		}
	}()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	go grpc.Init(sigCh)
	go http.Init(sigCh)
	listener.Init(appCtx)

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	appCancel()
	log.Logger.Infof("Received signal: %v, shutting down...", sig)
}
