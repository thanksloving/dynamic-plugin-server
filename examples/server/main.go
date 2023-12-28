package main

import (
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
	"github.com/thanksloving/dynamic-plugin-server/pkg/server"
	"net"

	log "github.com/sirupsen/logrus"

	_ "github.com/thanksloving/dynamic-plugin-server/repository"
)

func main() {
	dynamicService := server.NewDynamicService(pluggable.GetServiceDescriptors())
	lis, err := net.Listen("tcp", ":52051")
	if err != nil {
		panic(err)
	}
	log.Infof("server listening at %v", lis.Addr())
	if e := dynamicService.Start(lis); e != nil {
		panic(e)
	}
}
