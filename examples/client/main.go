package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/thanksloving/dynamic-plugin-server/client"
)

func main() {
	conn, err := grpc.DialContext(context.Background(), ":52051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	s := client.NewPluginStub(conn)
	// the request data is from the plugin meta info
	// s.GetPluginMetaList(context.Background(), &pb.MetaRequest{})
	result, err := s.Call(context.Background(), client.NewRequest("SayHello", map[string]any{"name": "plugin"}))
	if err != nil {
		panic(err)
	}
	log.Infof("got %+v", result)
}
