package server

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

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

func (ds *dynamicService) handler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
	method, namespace, pluginName, err := ds.router.GetMethodDesc(ctx)
	if err != nil {
		return nil, err
	}

	input := dynamicpb.NewMessage(method.Input())
	if err := dec(input); err != nil {
		return nil, err
	}

	req, err := protojson.Marshal(input)
	if err != nil {
		return nil, err
	}
	resp, err := pluggable.Call(ctx, namespace, pluginName, req)
	log.Infof("plugin request: %s, response: %s", string(req), string(resp))
	if err != nil {
		return nil, err
	}
	output := dynamicpb.NewMessage(method.Output())
	if err := protojson.Unmarshal(resp, output); err != nil {
		return nil, err
	}

	return output, nil
}

// GetPluginMetaList get plugin meta list
func (ds *dynamicService) GetPluginMetaList(_ context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	if request.Name != nil && request.Namespace == nil {
		return nil, status.Errorf(codes.InvalidArgument, "namespace is required")
	}
	return pluggable.GetPluginMetaList(request)
}
