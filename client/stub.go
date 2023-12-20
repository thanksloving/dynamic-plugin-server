package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/thanksloving/dynamic-plugin-server/pb"
	"github.com/thanksloving/dynamic-plugin-server/pluggable"
)

type (
	Stub interface {
		Call(ctx context.Context, serviceName string, methodName string, input map[string]any) ([]byte, error)
		GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error)
	}

	pluginStub struct {
		services   map[string]protoreflect.MethodDescriptor
		conn       *grpc.ClientConn
		metaClient pb.MetaServiceClient
	}
)

var _ Stub = &pluginStub{}

func NewPluginStub(conn *grpc.ClientConn, descriptor []protoreflect.ServiceDescriptor) Stub {
	ps := &pluginStub{
		services:   make(map[string]protoreflect.MethodDescriptor),
		conn:       conn,
		metaClient: pb.NewMetaServiceClient(conn),
	}

	ps.Parse(descriptor)
	return ps
}

func (ps *pluginStub) Call(ctx context.Context, serviceName string, methodName string, data map[string]any) ([]byte, error) {
	service := ps.services[getKey(serviceName, methodName)]
	if service == nil {
		return nil, errors.New("service not found")
	}

	input := dynamicpb.NewMessage(service.Input())
	fields := service.Input().Fields()
	for k, v := range data {
		name := fields.ByName(protoreflect.Name(k))
		input.Set(name, protoreflect.ValueOf(v))
	}

	output := dynamicpb.NewMessage(service.Output())

	method := fmt.Sprintf("/%s.%s/%s.%s.%s", pluggable.PackageName, serviceName, pluggable.DefaultNamespace, serviceName, methodName)
	err := ps.conn.Invoke(ctx, method, input, output)
	if err != nil {
		return nil, errors.Wrap(err, "invoke")
	}

	return protojson.Marshal(output)
}

func (ps *pluginStub) GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	return ps.metaClient.GetPluginMetaList(ctx, request)
}

func (ps *pluginStub) Parse(descriptors []protoreflect.ServiceDescriptor) {
	for _, descriptor := range descriptors {
		serviceName := descriptor.Name()
		for i := 0; i < descriptor.Methods().Len(); i++ {
			method := descriptor.Methods().Get(i)
			ps.services[getKey(serviceName, method.Name())] = method
			log.Infof("register service for client: %s.%s", serviceName, method.Name())
		}
	}
}

func getKey[T ~string](serviceName, methodName T) string {
	return strings.ToUpper(fmt.Sprintf("%s:%s", serviceName, methodName))
}
