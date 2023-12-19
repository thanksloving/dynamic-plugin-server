package pluggable

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

var instance = &registry{
	store: make(map[string]*pluggableInfo),
}

type (
	registry struct {
		store map[string]*pluggableInfo
		lock  sync.RWMutex
	}
	Option = func(*PluginMeta)
)

// Register register a pluggable service
func Register[I, O any](pluginName string, p Pluggable[I, O], opts ...Option) error {
	instance.lock.Lock()
	defer instance.lock.Unlock()

	meta := &PluginMeta{
		Namespace: DefaultNamespace,
		Name:      pluginName,
	}
	for _, opt := range opts {
		opt(meta)
	}

	key := instance.generateKey(meta.Namespace, pluginName)
	if _, ok := instance.store[key]; ok {
		return errors.Errorf("plugin %s already exists", key)
	}

	inputType := getGenericType[I]()
	outputType := getGenericType[O]()

	err := meta.parse(inputType, outputType)
	if err != nil {
		return err
	}

	instance.store[key] = &pluggableInfo{
		inputType:  inputType,
		outputType: outputType,
		meta:       meta,
		execute: func() func(ctx context.Context, param any) (any, error) {
			var limiter ratelimit.Limiter
			if meta.QPS != nil && *meta.QPS > 0 {
				limiter = ratelimit.New(*meta.QPS)
			}
			return func(ctx context.Context, param any) (_ any, err error) {
				defer func() {
					if r := recover(); r != nil {
						err = errors.Errorf("plugin %s, panic: %v", key, r)
						log.Errorf("param: %+v error: %v", param, err)
					}
				}()
				if limiter != nil {
					_ = limiter.Take()
				}
				return p.Execute(ctx, param.(I))
			}
		}(),
	}
	return nil
}

func getGenericType[T any]() reflect.Type {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func (r *registry) findPlugin(namespace, pluginName string) *pluggableInfo {
	instance.lock.RLock()
	defer instance.lock.RUnlock()
	key := r.generateKey(namespace, pluginName)
	return instance.store[key]
}

func (r *registry) generateKey(namespace, pluginName string) string {
	return strings.ToUpper(fmt.Sprintf("%s:%s", namespace, pluginName))
}
