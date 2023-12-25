package pluggable

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"

	"github.com/thanksloving/dynamic-plugin-server/pb"
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

	PluginDescriptor struct {
		p       *pluggableInfo
		input   *descriptorpb.DescriptorProto
		output  *descriptorpb.DescriptorProto
		service *descriptorpb.ServiceDescriptorProto
	}
)

func (pd *PluginDescriptor) getPluginMeta() *PluginMeta {
	return pd.p.meta
}

func (m *PluginMeta) Parse(p *pluggableInfo) (*PluginDescriptor, error) {
	service := &descriptorpb.ServiceDescriptorProto{
		Name: protoV2.String(m.Namespace),
		Method: []*descriptorpb.MethodDescriptorProto{
			{
				Name:       protoV2.String(m.Name),
				InputType:  protoV2.String(fmt.Sprintf(".%s.%s", m.Namespace, p.inputType.Name())),
				OutputType: protoV2.String(fmt.Sprintf(".%s.%s", m.Namespace, p.outputType.Name())),
			},
		},
	}
	inputMessage, err := m.resolveType(p.inputType, true)
	if err != nil {
		return nil, err
	}
	outputMessage, err := m.resolveType(p.outputType, false)
	if err != nil {
		return nil, err
	}
	return &PluginDescriptor{
		p:       p,
		input:   inputMessage,
		output:  outputMessage,
		service: service,
	}, nil
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
