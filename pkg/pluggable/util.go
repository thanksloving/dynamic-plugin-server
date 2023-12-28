package pluggable

import (
	"reflect"

	"github.com/golang/protobuf/ptypes/wrappers"
	protoV2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func getGenericType[T any]() reflect.Type {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
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
