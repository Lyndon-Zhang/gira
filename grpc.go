package gira

import "google.golang.org/grpc"

const GRPC_PATH_KEY = "girapath"

type GrpcServer interface {
	RegisterService(desc *grpc.ServiceDesc, impl interface{})
	GetServer(name string) (svr interface{}, ok bool)
}
