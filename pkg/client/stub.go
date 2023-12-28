package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/thanksloving/dynamic-plugin-server/pb"
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
)

type (
	Stub interface {
		Call(ctx context.Context, request *Request) ([]byte, error)
		GetPlugin(ctx context.Context, namespace, pluginName string) (*pluggable.PluginMeta, error)
		GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error)
	}

	pluginStub struct {
		services   map[string]protoreflect.MethodDescriptor
		conn       *grpc.ClientConn
		metaClient pb.MetaServiceClient
		version    string
		lock       sync.RWMutex
	}
)

var _ Stub = &pluginStub{}

func NewPluginStub(conn *grpc.ClientConn) Stub {
	ps := &pluginStub{
		services:   make(map[string]protoreflect.MethodDescriptor),
		conn:       conn,
		metaClient: pb.NewMetaServiceClient(conn),
	}
	ps.reloadAllServices()
	return ps
}

func (ps *pluginStub) reloadAllServices() {
	pageNum, pageSize := int32(1), int32(100)
	var descriptor []protoreflect.ServiceDescriptor
	var version string
	for {
		resp, err := ps.GetPluginMetaList(context.Background(), &pb.MetaRequest{Page: &pageNum, PageSize: &pageSize})
		if err != nil {
			log.Fatal(err)
		}
		version = lo.Ternary[string](pageNum == 1, resp.Version, version)

		// the server has changed when reloaded, do it again
		if version != resp.Version {
			pageNum = 1
		} else {
			// todo parse plugin descriptor to gRPC service descriptor
			//for _, plugin := range resp.Plugins {
			//	//descriptor = append(descriptor, pluggable.Parse(plugin))
			//}
			if resp.Total-pageSize*pageNum < pageSize {
				break
			}
			pageNum += 1
		}

	}

	// transform plugin meta to ServiceDescriptor
	ps.Parse(descriptor)
	return
}

func (ps *pluginStub) Parse(descriptors []protoreflect.ServiceDescriptor) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	for _, descriptor := range descriptors {
		serviceName := descriptor.Name()
		for i := 0; i < descriptor.Methods().Len(); i++ {
			method := descriptor.Methods().Get(i)
			ps.services[getKey(serviceName, method.Name())] = method
			log.Infof("register service for client: %s.%s", serviceName, method.Name())
		}
	}
}

func (ps *pluginStub) GetPlugin(ctx context.Context, namespace, pluginName string) (*pluggable.PluginMeta, error) {
	// todo get plugin meta from local cache, if version is not equal to server version, reload
	return nil, nil
}

func (ps *pluginStub) GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	resp, err := ps.metaClient.GetPluginMetaList(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "get plugin meta list")
	}
	if resp.Version != ps.version {
		// need to reload
		ps.reloadAllServices()
	}
	return resp, err
}

func (ps *pluginStub) Call(ctx context.Context, request *Request) ([]byte, error) {
	ps.lock.RLock()
	service := ps.services[getKey(request.getNamespace(), request.PluginName)]
	ps.lock.RUnlock()
	if service == nil {
		return nil, errors.New("service not found")
	}

	input := dynamicpb.NewMessage(service.Input())
	fields := service.Input().Fields()
	for k, v := range request.Data {
		name := fields.ByName(protoreflect.Name(k))
		input.Set(name, protoreflect.ValueOf(v))
	}

	output := dynamicpb.NewMessage(service.Output())

	err := ps.conn.Invoke(ctx, request.getGRpcMethodName(), input, output)
	if err != nil {
		return nil, errors.Wrap(err, "invoke")
	}

	return protojson.Marshal(output)
}

func getKey[T ~string](serviceName, methodName T) string {
	return strings.ToUpper(fmt.Sprintf("%s:%s", serviceName, methodName))
}
