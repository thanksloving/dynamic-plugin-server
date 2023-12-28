package client

import (
	"fmt"
	"time"

	"github.com/thanksloving/dynamic-plugin-server/pkg/macro"
)

type (
	Request interface {
		WithNamespace(namespace string) Request
		WithTimeout(timeout time.Duration) Request
	}

	request struct {
		Namespace  *string
		PluginName string
		Data       map[string]any
		Timeout    *time.Duration
	}
)

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

func (r *request) getNamespace() string {
	if r.Namespace != nil {
		return *r.Namespace
	}
	return macro.DefaultNamespace
}

// getGRpcMethodName build grpc method name, eg: /plugin_center.Default/Default.Default.SayHello
func (r *request) getGRpcMethodName() string {
	return fmt.Sprintf("/%s.%s/%s.%s.%s", macro.PackageName, r.getNamespace(), r.getNamespace(), r.getNamespace(), r.PluginName)
}
