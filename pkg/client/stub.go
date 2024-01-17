package client

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/thanksloving/dynamic-plugin-server/pb"
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
)

type (
	Stub interface {
		Call(ctx context.Context, request Request) ([]byte, error)
		GetPlugin(ctx context.Context, namespace, pluginName string) (*pluggable.PluginMeta, error)
		GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error)
	}

	pluginStub struct {
		conn   *grpc.ClientConn
		router *router
	}
)

var _ Stub = &pluginStub{}

func NewPluginStub(conn *grpc.ClientConn) Stub {
	router := newRouter(conn)
	ps := &pluginStub{
		conn:   conn,
		router: router,
	}
	return ps
}

func (ps *pluginStub) GetPlugin(ctx context.Context, namespace, pluginName string) (*pluggable.PluginMeta, error) {
	// todo get plugin meta from local cache, if version is not equal to server version, reload
	return nil, nil
}

func (ps *pluginStub) GetPluginMetaList(ctx context.Context, request *pb.MetaRequest) (*pb.MetaResponse, error) {
	return ps.router.getPluginMetaList(ctx, request)
}

func (ps *pluginStub) Call(ctx context.Context, request Request) ([]byte, error) {
	service := ps.router.getMethodDescriptor(request.GetNamespace(), request.GetPluginName())
	if service == nil {
		return nil, errors.New("service not found")
	}

	input := request.AssembleRequestMessage(service.Input())
	output := dynamicpb.NewMessage(service.Output())

	if timeout := request.GetTimeout(); timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	err := ps.conn.Invoke(ctx, request.GetGRpcMethodName(), input, output)
	if err != nil {
		return nil, errors.Wrap(err, "invoke")
	}

	return protojson.Marshal(output)
}
