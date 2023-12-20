package pluggable

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"

	"github.com/thanksloving/dynamic-plugin-server/pb"
)

var instance = &registry{
	store:   make(map[string]*pluggableInfo),
	version: time.Now().Format("20060102150405"),
}

type (
	registry struct {
		store   map[string]*pluggableInfo
		lock    sync.RWMutex
		version string
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

func GetPluginMetaList(request *pb.MetaRequest) (*pb.MetaResponse, error) {
	page := lo.Ternary[int](request.Page == nil, 1, int(*request.Page))
	size := lo.Ternary[int](request.PageSize == nil, 20, int(*request.PageSize))
	instance.lock.RLock()
	defer instance.lock.RUnlock()
	if request.Name != nil {
		if info := instance.findPlugin(*request.Namespace, *request.Name); info != nil {
			return &pb.MetaResponse{
				Total: 1,
				Plugins: []*pb.PluginMeta{
					info.transform(),
				},
				Version: instance.version,
			}, nil
		}
		return nil, errors.Errorf("plugin %s not found", *request.Name)
	}
	var plugins = instance.store
	if request.Namespace != nil {
		plugins = lo.OmitBy[string, *pluggableInfo](instance.store, func(key string, p *pluggableInfo) bool {
			return p.meta.Namespace != request.GetNamespace()
		})
	}
	list := lo.MapToSlice[string, *pluggableInfo, *pb.PluginMeta](plugins, func(key string, p *pluggableInfo) *pb.PluginMeta {
		return &pb.PluginMeta{
			Name:      p.meta.Name,
			Namespace: p.meta.Namespace,
			Desc:      p.meta.Desc,
			Timeout:   p.meta.Timeout,
			CacheTime: p.meta.CacheTime,
			Input:     p.meta.transformInput(),
			Output:    p.meta.transformOutput(),
		}
	})
	total := len(list)
	var start, end int
	start = lo.Ternary[int]((page-1)*size < total, (page-1)*size, total)
	end = lo.Ternary[int](start+size <= total, start+size, total)
	list = list[start:end]
	return &pb.MetaResponse{
		Total:   int32(total),
		Plugins: list,
		Version: instance.version,
	}, nil
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
