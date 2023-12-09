package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/thanksloving/dynamic-plugin-server/client"
	"github.com/thanksloving/dynamic-plugin-server/pluggable"
)

func TestPluginStub_Call(t *testing.T) {
	conn, err := grpc.DialContext(context.Background(), ":52051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.Equal(t, false, conn == nil)
	assert.NoError(t, err)
	s := client.NewPluginStub(conn, pluggable.GetServiceDescriptors())
	r, e := s.Call(context.Background(), pluggable.DefaultNamespace, "SayHello", map[string]any{"name": "plugin"})
	assert.NoError(t, e)
	assert.Equal(t, []byte(`{"message":"hello plugin"}`), r)
}
