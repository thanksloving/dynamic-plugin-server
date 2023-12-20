package pluggable

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/samber/lo"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/thanksloving/dynamic-plugin-server/pb"
)

var once sync.Once

var (
	messageTypes []*descriptorpb.DescriptorProto
	services     []*descriptorpb.ServiceDescriptorProto
)

type (
	PluginMeta struct {
		Name      string
		QPS       *int
		Namespace string // service name
		Desc      string
		Timeout   *int64
		CacheTime *int64
		Inputs    []Input
		Outputs   []Output
	}
)

// FIXME: 生成服务描述符
func (m *PluginMeta) parse(input, output reflect.Type) error {
	service := &descriptorpb.ServiceDescriptorProto{
		Name: protoV2.String(m.Namespace),
		Method: []*descriptorpb.MethodDescriptorProto{
			{
				Name:       protoV2.String(m.Name),
				InputType:  protoV2.String(fmt.Sprintf(".%s.%s", m.Namespace, input.Name())),
				OutputType: protoV2.String(fmt.Sprintf(".%s.%s", m.Namespace, output.Name())),
			},
		},
	}
	inputMessage, err := m.resolveType(input, true)
	if err != nil {
		return err
	}
	outputMessage, err := m.resolveType(output, false)
	if err != nil {
		return err
	}

	messageTypes = append(messageTypes, inputMessage, outputMessage)
	services = append(services, service)
	return nil
}

func (m *PluginMeta) resolveType(t reflect.Type, isInput bool) (*descriptorpb.DescriptorProto, error) {
	desc := &descriptorpb.DescriptorProto{
		Name: protoV2.String(t.Name()),
	}
	optional := false
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		optional = true
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		desc.Field = append(desc.Field, &descriptorpb.FieldDescriptorProto{
			Name:   protoV2.String(field.Name),
			Number: protoV2.Int32(int32(i + 1)),
			Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:   m.getFieldType(field.Type),
		})
		item := Item{
			Name: field.Name,
			Type: field.Type.String(),
			Desc: field.Tag.Get("desc"),
		}
		if isInput {
			var options []any
			if optionStr := field.Tag.Get("options"); optionStr != "" {
				if err := sonic.Unmarshal([]byte(optionStr), &options); err != nil {
					return nil, err
				}
			}
			m.Inputs = append(m.Inputs, Input{Item: item, Optional: optional, Options: options})
		} else {
			m.Outputs = append(m.Outputs, Output{Item: item})
		}
	}

	return desc, nil
}

func (m *PluginMeta) getFieldType(t reflect.Type) *descriptorpb.FieldDescriptorProto_Type {
	switch t.Kind() {
	case reflect.String:
		return descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()
	case reflect.Bool:
		return descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()
	case reflect.Int:
		return descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()
	case reflect.Int64:
		return descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()
	case reflect.Float32, reflect.Float64:
		return descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum()
	case reflect.Uint, reflect.Uint32:
		return descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum()
	case reflect.Uint64:
		return descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()
	default:
		// TODO add other types and nested types
		return descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum()
	}
}

func GetServices() []protoreflect.ServiceDescriptor {
	file := &descriptorpb.FileDescriptorProto{
		Syntax:      protoV2.String("proto3"),
		Name:        protoV2.String("services.proto"),
		Package:     protoV2.String(PackageName),
		MessageType: messageTypes,
		Service:     services,
	}
	fds, _ := protodesc.NewFile(file, nil)
	var sds []protoreflect.ServiceDescriptor
	for i := 0; i < fds.Services().Len(); i++ {
		sds = append(sds, fds.Services().Get(i))
	}
	return sds
}

func GetServiceDescriptors() []protoreflect.ServiceDescriptor {
	once.Do(func() {
		// generate service descriptors
	})
	// 创建一个文件描述符定义
	file := &descriptorpb.FileDescriptorProto{
		Syntax:  protoV2.String("proto3"),
		Name:    protoV2.String("service.proto"),
		Package: protoV2.String("plugin_center"),
		// 描述消息
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoV2.String("Empty"),
			},
			{
				Name: protoV2.String("HelloRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   protoV2.String("name"),
						Number: protoV2.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			{
				Name: protoV2.String("HelloReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   protoV2.String("message"),
						Number: protoV2.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: protoV2.String(DefaultNamespace),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       protoV2.String("SayHello"),
						InputType:  protoV2.String(".plugin_center.HelloRequest"),
						OutputType: protoV2.String(".plugin_center.HelloReply"),
					},
				},
			},
		},
	}

	fdSet := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{file},
	}

	files, err := protodesc.NewFiles(fdSet)
	if err != nil {
		log.Fatalf("Failed creating new file descriptor set: %v", err)
	}

	var sds []protoreflect.ServiceDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			sds = append(sds, fd.Services().Get(i))
		}
		return true
	})

	return sds
}

func (m *PluginMeta) transformInput() []*pb.PluginMeta_Input {
	return lo.Map[Input, *pb.PluginMeta_Input](m.Inputs, func(item Input, index int) *pb.PluginMeta_Input {
		return &pb.PluginMeta_Input{
			Name:     item.Name,
			Type:     item.Type,
			Desc:     item.Desc,
			Required: !item.Optional,
			Options: lo.Map[any, *anypb.Any](item.Options, func(item any, index int) *anypb.Any {
				a, _ := convertInterfaceToAny(item)
				return a
			}),
		}
	})
}

func (m *PluginMeta) transformOutput() []*pb.PluginMeta_Output {
	return lo.Map[Output, *pb.PluginMeta_Output](m.Outputs, func(item Output, index int) *pb.PluginMeta_Output {
		return &pb.PluginMeta_Output{
			Name: item.Name,
			Type: item.Type,
			Desc: item.Desc,
		}
	})
}

func convertInterfaceToAny(v interface{}) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	bytes, err := defaultCodec.Marshal(v)
	if err != nil {
		return nil, err
	}
	bytesValue := &wrappers.BytesValue{
		Value: bytes,
	}
	err = anypb.MarshalFrom(anyValue, bytesValue, protoV2.MarshalOptions{})
	return anyValue, err
}
