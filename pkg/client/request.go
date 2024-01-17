package client

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/thanksloving/dynamic-plugin-server/pkg/macro"
)

type (
	Request interface {
		WithNamespace(namespace string) Request
		WithTimeout(timeout time.Duration) Request

		GetPluginName() string
		GetNamespace() string
		GetTimeout() *time.Duration
		GetGRpcMethodName() string

		// AssembleRequestMessage assemble request message by request data
		AssembleRequestMessage(md protoreflect.MessageDescriptor) *dynamicpb.Message
	}

	request struct {
		Namespace  *string
		PluginName string
		Data       map[string]any
		Timeout    *time.Duration
	}
)

// NewRequest create a new request, by map[string]any, your can implement your own request to overwrite the GetRequestMessage method
func NewRequest(pluginName string, data map[string]any) Request {
	return &request{
		PluginName: pluginName,
		Data:       data,
	}
}

func (r *request) WithNamespace(namespace string) Request {
	r.Namespace = &namespace
	return r
}

func (r *request) WithTimeout(timeout time.Duration) Request {
	r.Timeout = &timeout
	return r
}

func (r *request) GetTimeout() *time.Duration {
	return r.Timeout
}

func (r *request) GetNamespace() string {
	if r.Namespace != nil {
		return *r.Namespace
	}
	return macro.DefaultNamespace
}

func (r *request) GetPluginName() string {
	return r.PluginName
}

func (r *request) AssembleRequestMessage(md protoreflect.MessageDescriptor) *dynamicpb.Message {
	input := dynamicpb.NewMessage(md)
	fields := md.Fields()
	for k, v := range r.Data {
		name := fields.ByName(protoreflect.Name(k))
		input.Set(name, protoreflect.ValueOf(v))
	}
	return input
}

// GetGRpcMethodName build grpc method name, eg: /plugin_center.Default/Default.Default.SayHello
func (r *request) GetGRpcMethodName() string {
	return fmt.Sprintf("/%s.%s/%s.%s.%s", macro.PackageName, r.GetNamespace(), r.GetNamespace(), r.GetNamespace(), r.PluginName)
}
