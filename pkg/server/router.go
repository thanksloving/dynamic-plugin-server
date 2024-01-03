package server

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
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
			gsd.Methods = append(gsd.Methods, grpc.MethodDesc{MethodName: string(method.FullName()), Handler: ds.handler})
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
