package client

import (
	"fmt"
	"sync"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/common/__PROTO_PACKAGE__"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	conn           *grpc.ClientConn
	client         __PROTO_PACKAGE__.X_GRPC_SERVICE__Client
	clientSyncOnce sync.Once
)

func GetX_GRPC_SERVICE__Client(config *GRpcClientConfig) (__PROTO_PACKAGE__.X_GRPC_SERVICE__Client, error) {
	clientSyncOnce.Do(func() {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(1024 * 1024)),
		}
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", config.Host, config.Port), opts...)
		if err != nil {
			panic(err)
		}
		client = __PROTO_PACKAGE__.NewX_GRPC_SERVICE__Client(conn)
	})
	return client, nil
}

func Destroy() {
	if conn != nil {
		err := conn.Close()
		if err != nil {
			fmt.Printf("Failed to close gRPC connection: %v\n", err)
		}
	}
}
