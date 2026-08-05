package grpc

import (
	"context"

	"github.com/nusiss-capstone-project/asset-mservice/common/assetpb"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
)

type AssetService struct {
	assetpb.UnimplementedAssetServiceServer
}

func (s *AssetService) SayHello(ctx context.Context, in *assetpb.HelloRequest) (*assetpb.HelloResponse, error) {
	log.Logger.Infof("Received: %v", in.GetName())
	return &assetpb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}
