package pluggable

import (
	"context"
	"reflect"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"

	"github.com/thanksloving/dynamic-plugin-server/pb"
)

type pluggableInfo struct {
	execute    func(ctx context.Context, param any) (any, error)
	inputType  reflect.Type
	outputType reflect.Type
	meta       *PluginMeta
}

func Call(ctx context.Context, namespace, pluginName string, input []byte) ([]byte, error) {
	plugin := instance.findPlugin(namespace, pluginName)
	if plugin == nil {
		return nil, errors.Errorf("plugin %s:%s not found", namespace, pluginName)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, plugin.getTimeout())
	defer cancel()

	param := reflect.New(plugin.inputType).Interface()
	if err := sonic.Unmarshal(input, param); err != nil {
		return nil, err
	}
	// todo check param

	return plugin.run(ctx, param, func() ([]byte, error) {
		result, err := plugin.execute(ctx, param)
		if err != nil {
			return nil, err
		}
		return sonic.Marshal(result)
	})
}

func (p *pluggableInfo) getTimeout() time.Duration {
	timeout := defaultTimeout
	if p.meta.Timeout != nil && *p.meta.Timeout > 0 {
		timeout = time.Duration(*p.meta.Timeout) * time.Millisecond
	}
	return timeout
}

func (p *pluggableInfo) run(ctx context.Context, param any, execute func() ([]byte, error)) ([]byte, error) {
	if p.meta.CacheTime == nil || *p.meta.CacheTime <= 0 {
		return execute()
	}
	cacheTime := time.Duration(*p.meta.CacheTime) * time.Millisecond
	var cacheKey string
	if ck, ok := param.(CustomCacheKey); ok {
		cacheKey = ck.GenerateKey(p.meta.Namespace, p.meta.Name)
	} else {
		// todo generate key
	}

	if result, _ := defaultCache.Get(ctx, cacheKey); result != nil {
		return result.([]byte), nil
	}
	result, err := execute()
	go func() {
		_ = defaultCache.Set(context.Background(), cacheKey, result, cacheTime)
	}()
	return result, err
}

func (p *pluggableInfo) transform() *pb.PluginMeta {
	return &pb.PluginMeta{
		Namespace: p.meta.Namespace,
		Name:      p.meta.Name,
		Input:     p.meta.transformInput(),
		Output:    p.meta.transformOutput(),
	}
}
