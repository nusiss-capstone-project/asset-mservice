package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/config"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/grpc"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/telemetry"
)

var (
	sigCh = make(chan os.Signal, 1)
)

func main() {
	config.Init()
	log.InitLogger()
	repository.Init()

	shutdownTelemetry := telemetry.Init(context.Background())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			log.Logger.Errorw("telemetry shutdown failed", "error", err)
		}
	}()

	go grpc.Init(sigCh)
	go http.Init(sigCh)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Logger.Infof("Received signal: %v, shutting down...", sig)
}
