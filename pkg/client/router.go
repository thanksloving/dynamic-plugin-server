package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/thanksloving/dynamic-plugin-server/pb"
)

type router struct {
	services   map[string]protoreflect.MethodDescriptor
	metaClient pb.MetaServiceClient
	lock       sync.RWMutex
	version    string
}

func newRouter(conn *grpc.ClientConn) *router {
	r := &router{
		metaClient: pb.NewMetaServiceClient(conn),
	}
	r.init()
	return r
}

func (r *router) init() {
	r.services = make(map[string]protoreflect.MethodDescriptor)
	pageNum, pageSize := int32(1), int32(100)
	var descriptor []protoreflect.ServiceDescriptor
	var version string
	for {
		resp, err := r.getPluginMetaList(context.Background(), &pb.MetaRequest{Page: &pageNum, PageSize: &pageSize})
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
	r.Parse(descriptor)
}

func (r *router) getMethodDescriptor(serviceName, pluginName string) protoreflect.MethodDescriptor {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.services[getKey(serviceName, pluginName)]
}

func (r *router) getPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	return r.metaClient.GetPluginMetaList(ctx, request)
}

func (r *router) Parse(descriptors []protoreflect.ServiceDescriptor) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, descriptor := range descriptors {
		serviceName := descriptor.Name()
		for i := 0; i < descriptor.Methods().Len(); i++ {
			method := descriptor.Methods().Get(i)
			r.services[getKey(serviceName, method.Name())] = method
			log.Infof("register service for client: %s.%s", serviceName, method.Name())
		}
	}
}

func getKey[T ~string](serviceName, methodName T) string {
	return strings.ToUpper(fmt.Sprintf("%s:%s", serviceName, methodName))
}
