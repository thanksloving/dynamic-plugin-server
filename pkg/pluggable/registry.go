package pluggable

import (
	"context"
	"fmt"
	"github.com/thanksloving/dynamic-plugin-server/pkg/macro"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/thanksloving/dynamic-plugin-server/pb"
)

var instance = &registry{
	store:   make(map[string]*pluggableInfo),
	version: time.Now().Format("20060102150405"),
}

type (
	registry struct {
		store map[string]*pluggableInfo
		lock  sync.RWMutex

		pluginDescriptors []*PluginDescriptor

		services []protoreflect.ServiceDescriptor
		version  string
	}
	Option = func(*PluginMeta)
)

// Register register a pluggable service
func Register[I, O any](pluginName string, p Pluggable[I, O], opts ...Option) error {
	instance.lock.Lock()
	defer instance.lock.Unlock()

	meta := &PluginMeta{
		Namespace: macro.DefaultNamespace,
		Name:      pluginName,
	}
	for _, opt := range opts {
		opt(meta)
	}

	key := instance.generateKey(meta.Namespace, pluginName)
	if _, ok := instance.store[key]; ok {
		return errors.Errorf("plugin %s already exists", key)
	}

	info := &pluggableInfo{
		inputType:  getGenericType[I](),
		outputType: getGenericType[O](),
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
	if err := info.apply(instance); err != nil {
		return err
	}
	instance.store[key] = info
	instance.version = time.Now().Format("20060102150405")
	return nil
}

func Unregister(namespace, pluginName string) bool {
	instance.lock.Lock()
	defer instance.lock.Unlock()

	key := instance.generateKey(namespace, pluginName)
	if _, ok := instance.store[key]; !ok {
		return false
	}
	delete(instance.store, key)
	for i, descriptor := range instance.pluginDescriptors {
		if descriptor.getPluginMeta().Name == pluginName && descriptor.getPluginMeta().Namespace == namespace {
			instance.pluginDescriptors = append(instance.pluginDescriptors[:i], instance.pluginDescriptors[i+1:]...)
			break
		}
	}

	instance.version = time.Now().Format("20060102150405")
	return true
}

func (*registry) appendDescriptor(descriptor *PluginDescriptor) {
	instance.lock.Lock()
	defer instance.lock.Unlock()

	instance.pluginDescriptors = append(instance.pluginDescriptors, descriptor)
}

func (*registry) generateKey(namespace, pluginName string) string {
	return strings.ToUpper(fmt.Sprintf("%s:%s", namespace, pluginName))
}

func findPlugin(namespace, pluginName string) *pluggableInfo {
	instance.lock.RLock()
	defer instance.lock.RUnlock()
	key := instance.generateKey(namespace, pluginName)
	return instance.store[key]
}

// GetServiceDescriptors get all service descriptors
func GetServiceDescriptors() []protoreflect.ServiceDescriptor {
	var messageTypes []*descriptorpb.DescriptorProto
	var services []*descriptorpb.ServiceDescriptorProto
	for _, pluginDescriptor := range instance.pluginDescriptors {
		services = append(services, pluginDescriptor.service)
		messageTypes = append(messageTypes, pluginDescriptor.input, pluginDescriptor.output)
	}
	file := &descriptorpb.FileDescriptorProto{
		Syntax:      protoV2.String("proto3"),
		Name:        protoV2.String("services.proto"),
		Package:     protoV2.String(macro.PackageName),
		MessageType: messageTypes,
		Service:     services,
	}
	fds, _ := protodesc.NewFile(file, nil)
	var sds []protoreflect.ServiceDescriptor
	for i := 0; i < fds.Services().Len(); i++ {
		sds = append(sds, fds.Services().Get(i))
	}
	instance.version = time.Now().Format("20060102150405")
	instance.services = sds
	return sds
}

// GetPluginMetaList get all plugin meta, for client query
func GetPluginMetaList(request *pb.MetaRequest) (*pb.MetaResponse, error) {
	page := lo.Ternary[int](request.Page == nil, 1, int(*request.Page))
	size := lo.Ternary[int](request.PageSize == nil, 20, int(*request.PageSize))
	if request.Name != nil {
		if info := findPlugin(*request.Namespace, *request.Name); info != nil {
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
	instance.lock.RLock()
	defer instance.lock.RUnlock()
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
