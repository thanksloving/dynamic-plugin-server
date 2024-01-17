package server

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Router interface {
		GetMethodDesc(ctx context.Context) (*PluginService, error)
		GetServiceDescList() []*grpc.ServiceDesc
	}

	serviceRouter struct {
		services map[string]PluginService
	}

	PluginService struct {
		Method      protoreflect.MethodDescriptor
		ServiceName string
		PluginName  string
	}
)

func newServiceRouter(serviceDescriptions []protoreflect.ServiceDescriptor) Router {
	s := &serviceRouter{}
	s.services = s.resolveServices(serviceDescriptions)
	return s
}

func (s *serviceRouter) GetServiceDescList() []*grpc.ServiceDesc {
	var serviceDescList []*grpc.ServiceDesc
	for fullName := range s.services {
		gsd := grpc.ServiceDesc{ServiceName: fullName, HandlerType: (*any)(nil)}
		gsd.Methods = append(gsd.Methods, grpc.MethodDesc{MethodName: fullName, Handler: nil})
		serviceDescList = append(serviceDescList, &gsd)
	}
	return serviceDescList
}

func (s *serviceRouter) resolveServices(serviceDescriptions []protoreflect.ServiceDescriptor) map[string]PluginService {
	services := make(map[string]PluginService)
	for _, sd := range serviceDescriptions {
		for idx := 0; idx < sd.Methods().Len(); idx++ {
			method := sd.Methods().Get(idx)
			services[string(method.FullName())] = PluginService{
				Method:      method,
				ServiceName: string(sd.FullName()),
				PluginName:  string(method.FullName()),
			}
		}
	}
	return services
}

// GetMethodDesc get method descriptor,if the service is offline in the runtime, return error
func (s *serviceRouter) GetMethodDesc(ctx context.Context) (*PluginService, error) {
	stream := grpc.ServerTransportStreamFromContext(ctx)
	//e.g. stream method:/plugin_center.Default/plugin_center.Default.SayHello
	if idx := strings.LastIndex(stream.Method(), "/"); idx != -1 {
		key := stream.Method()[idx+1:]
		if methods := strings.Split(key, "."); len(methods) == 3 {
			if pluginService, ok := s.services[key]; ok {
				return &pluginService, nil
			}
		}
	}
	return nil, status.Errorf(codes.NotFound, "Unknown plugin, %s", stream.Method())
}
