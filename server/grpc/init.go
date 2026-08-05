package grpc

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/common/__PROTO_PACKAGE__"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/config"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpcpkg "google.golang.org/grpc"
)

func Init(exitSig chan os.Signal) {
	ipPort := fmt.Sprintf("%s:%d", config.Config.GrpcConfig.Host, config.Config.GrpcConfig.Port)
	listener, err := net.Listen("tcp", ipPort)
	if err != nil {
		log.Logger.Fatalf("Failed to listen: %v", err)
		exitSig <- os.Interrupt
		return
	}
	// Set up gRPC options for timeout and connection pooling
	opts := []grpcpkg.ServerOption{
		grpcpkg.ConnectionTimeout(time.Duration(config.Config.GrpcConfig.ConnectTimeout) * time.Second),
		grpcpkg.MaxConcurrentStreams(uint32(config.Config.GrpcConfig.MaxPoolSize)),
		grpcpkg.MaxRecvMsgSize(1024 * 1024),
		grpcpkg.MaxSendMsgSize(1024 * 1024),
		grpcpkg.StatsHandler(otelgrpc.NewServerHandler()),
	}
	grpcServer := grpcpkg.NewServer(opts...)
	__PROTO_PACKAGE__.RegisterX_GRPC_SERVICE__Server(grpcServer, &X_GRPC_SERVICE__{})

	log.Logger.Infof("Server is running on %s", ipPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Logger.Fatal("Failed to serve: %v", err)
		exitSig <- os.Interrupt
	}
}
