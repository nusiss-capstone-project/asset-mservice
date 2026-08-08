package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/common/assetpb"
	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Init(exitSig chan os.Signal) {
	ipPort := fmt.Sprintf("%s:%d", config.Config.GrpcConfig.Host, config.Config.GrpcConfig.Port)
	listener, err := net.Listen("tcp", ipPort)
	if err != nil {
		log.Logger.Fatalf("Failed to listen: %v", err)
		exitSig <- os.Interrupt
		return
	}
	opts := []grpcpkg.ServerOption{
		grpcpkg.ConnectionTimeout(time.Duration(config.Config.GrpcConfig.ConnectTimeout) * time.Second),
		grpcpkg.MaxConcurrentStreams(uint32(config.Config.GrpcConfig.MaxPoolSize)),
		grpcpkg.MaxRecvMsgSize(1024 * 1024),
		grpcpkg.MaxSendMsgSize(1024 * 1024),
		grpcpkg.StatsHandler(otelgrpc.NewServerHandler()),
		grpcpkg.ChainUnaryInterceptor(unaryLoggingInterceptor()),
	}
	grpcServer := grpcpkg.NewServer(opts...)
	assetpb.RegisterAssetServiceServer(grpcServer, &AssetService{})

	log.Logger.Infow("grpc server is running", "addr", ipPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Logger.Fatal("Failed to serve: %v", err)
		exitSig <- os.Interrupt
	}
}

func unaryLoggingInterceptor() grpcpkg.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpcpkg.UnaryServerInfo,
		handler grpcpkg.UnaryHandler,
	) (any, error) {
		start := time.Now()
		log.WithContext(ctx).Infow("grpc request started", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		durationMs := float64(time.Since(start).Microseconds()) / 1000
		fields := []any{
			"method", info.FullMethod,
			"duration_ms", durationMs,
			"code", status.Code(err).String(),
		}
		if err != nil {
			log.WithContext(ctx).Errorw("grpc request failed", append(fields, "error", err)...)
			return resp, err
		}
		log.WithContext(ctx).Infow("grpc request completed", fields...)
		return resp, nil
	}
}
