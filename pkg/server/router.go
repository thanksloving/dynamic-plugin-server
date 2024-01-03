package server

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type (
	Router interface {
		GetMethodDesc(ctx context.Context) (method protoreflect.MethodDescriptor, serviceName string, pluginName string, err error)
	}

	serviceRouter struct {
		methods map[string]protoreflect.MethodDescriptor
	}
)

func newServiceRouter(server *grpc.Server, serviceDescriptions []protoreflect.ServiceDescriptor) Router {
	s := &serviceRouter{
		methods: make(map[string]protoreflect.MethodDescriptor),
	}
	serviceDescList := s.resolveServices(serviceDescriptions)
	for _, sd := range serviceDescList {
		server.RegisterService(sd, s)
	}
	return s
}

func (s *serviceRouter) resolveServices(serviceDescriptions []protoreflect.ServiceDescriptor) []*grpc.ServiceDesc {
	var serviceDescList []*grpc.ServiceDesc
	for _, sd := range serviceDescriptions {
		gsd := grpc.ServiceDesc{ServiceName: string(sd.FullName()), HandlerType: (*any)(nil)}
		for idx := 0; idx < sd.Methods().Len(); idx++ {
			method := sd.Methods().Get(idx)
			gsd.Methods = append(gsd.Methods, grpc.MethodDesc{MethodName: string(method.FullName()), Handler: s.handler})
			s.methods[string(method.FullName())] = method
			log.Infof("register service: %s", string(method.FullName()))
		}
		serviceDescList = append(serviceDescList, &gsd)
	}
	return serviceDescList
}

// GetMethodDesc get method descriptor,if the service is offline in the runtime, return error
func (s *serviceRouter) GetMethodDesc(ctx context.Context) (method protoreflect.MethodDescriptor, serviceName string, pluginName string, err error) {
	stream := grpc.ServerTransportStreamFromContext(ctx)
	//e.g. stream method:/plugin_center.Default/plugin_center.Default.SayHello
	if idx := strings.LastIndex(stream.Method(), "/"); idx != -1 {
		key := stream.Method()[idx+1:]
		if methods := strings.Split(key, "."); len(methods) == 3 {
			if method = s.methods[key]; method != nil {
				serviceName = methods[1]
				pluginName = methods[2]
				return
			}
		}
	}
	err = status.Errorf(codes.NotFound, "Unknown plugin, %s", stream.Method())
	return
}

func (s *serviceRouter) handler(_ any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
	method, namespace, pluginName, err := s.GetMethodDesc(ctx)
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
