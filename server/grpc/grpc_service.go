package grpc

import (
	"context"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/common/__PROTO_PACKAGE__"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
)

type X_GRPC_SERVICE__ struct {
	__PROTO_PACKAGE__.UnimplementedX_GRPC_SERVICE__Server
}

func (s *X_GRPC_SERVICE__) SayHello(ctx context.Context, in *__PROTO_PACKAGE__.HelloRequest) (*__PROTO_PACKAGE__.HelloResponse, error) {
	log.Logger.Infof("Received: %v", in.GetName())
	return &__PROTO_PACKAGE__.HelloResponse{Message: "Hello " + in.GetName()}, nil
}
