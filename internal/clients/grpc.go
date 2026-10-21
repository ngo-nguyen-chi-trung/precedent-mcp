package clients

import (
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ngo/precedent_mcp/internal/env"
)

var (
	grpcConnectionInstance *grpc.ClientConn
	grpcConnectionOnce     sync.Once
)

func GetGRPCConnectionInstance() *grpc.ClientConn {
	grpcConnectionOnce.Do(func() {
		conn, err := grpc.NewClient(env.GRPCAddress.GetValue(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			panic(err)
		}
		grpcConnectionInstance = conn
	})
	return grpcConnectionInstance
}
