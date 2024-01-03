package server

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/thanksloving/dynamic-plugin-server/pb"
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
	_ "github.com/thanksloving/dynamic-plugin-server/repository"
)

type (
	dynamicService struct {
		router Router
		server *grpc.Server
		pb.MetaServiceServer
	}

	DynamicService interface {
		Start(listener net.Listener) error
	}
)

func NewDynamicService(options ...grpc.ServerOption) DynamicService {
	server := grpc.NewServer(options...)

	router := newServiceRouter(server, pluggable.GetRegistryServiceDescriptors())
	ds := &dynamicService{
		router: router,
		server: server,
	}
	// register meta service
	pb.RegisterMetaServiceServer(server, ds)
	reflection.Register(server)
	return ds
}

func (ds *dynamicService) Start(listener net.Listener) error {
	return ds.server.Serve(listener)
}

// GetPluginMetaList get plugin meta list
func (ds *dynamicService) GetPluginMetaList(_ context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	if request.Name != nil && request.Namespace == nil {
		return nil, status.Errorf(codes.InvalidArgument, "namespace is required")
	}
	return pluggable.GetPluginMetaList(request)
}
