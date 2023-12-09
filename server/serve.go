package server

import (
	"context"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/thanksloving/dynamic-plugin-server/pluggable"
)

type (
	dynamicService struct {
		methods map[string]protoreflect.MethodDescriptor
		server  *grpc.Server
	}

	DynamicService interface {
		Start(listener net.Listener) error
	}
)

func NewDynamicService(fileDescriptions []protoreflect.ServiceDescriptor, options ...grpc.ServerOption) DynamicService {
	ds := &dynamicService{
		methods: make(map[string]protoreflect.MethodDescriptor),
	}

	server := grpc.NewServer(options...)
	descList := ds.resolveServices(fileDescriptions)
	for _, serviceDesc := range descList {
		server.RegisterService(serviceDesc, ds)
	}
	reflection.Register(server)
	ds.server = server
	return ds
}

func (ds *dynamicService) Start(listener net.Listener) error {
	return ds.server.Serve(listener)
}

func (ds *dynamicService) resolveServices(serviceDescriptions []protoreflect.ServiceDescriptor) []*grpc.ServiceDesc {
	var serviceDescList []*grpc.ServiceDesc
	for _, sd := range serviceDescriptions {
		gsd := grpc.ServiceDesc{ServiceName: string(sd.FullName()), HandlerType: (*interface{})(nil)}
		for idx := 0; idx < sd.Methods().Len(); idx++ {
			method := sd.Methods().Get(idx)
			gsd.Methods = append(gsd.Methods, grpc.MethodDesc{MethodName: string(method.FullName()), Handler: ds.handler})
			ds.methods[string(method.FullName())] = method
			log.Infof("register service: %s", string(method.FullName()))
		}
		serviceDescList = append(serviceDescList, &gsd)
	}
	return serviceDescList
}

func (ds *dynamicService) getMethodDesc(ctx context.Context) (method protoreflect.MethodDescriptor, serviceName string, pluginName string, err error) {
	stream := grpc.ServerTransportStreamFromContext(ctx)
	//eg. stream method:/plugin_center.Default/plugin_center.Default.SayHello
	if idx := strings.LastIndex(stream.Method(), "/"); idx != -1 {
		key := stream.Method()[idx+1:]
		if methods := strings.Split(key, "."); len(methods) == 3 {
			if method = ds.methods[key]; method != nil {
				serviceName = methods[1]
				pluginName = methods[2]
				return
			}
		}
	}
	err = status.Errorf(codes.NotFound, "Unknown plugin, %s", stream.Method())
	return
}

func (ds *dynamicService) handler(_ interface{}, ctx context.Context, dec func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
	method, namespace, pluginName, err := ds.getMethodDesc(ctx)
	if err != nil {
		return nil, err
	}

	if namespace == "MetaService" {
		return ds.Meta(ctx, pluginName)
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
