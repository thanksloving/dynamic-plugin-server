package pluggable

import "github.com/vmihailenco/msgpack/v5"

type (
	Codec interface {
		// Marshal returns the wire format of v.
		Marshal(v interface{}) ([]byte, error)
		// Unmarshal parses the wire format into v.
		Unmarshal(data []byte, v interface{}) error
	}
	MsgpackCodec struct{}
)

func (*MsgpackCodec) Name() string {
	return "msgpack"
}

func (*MsgpackCodec) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (*MsgpackCodec) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
