package main

import (
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/thanksloving/dynamic-plugin-server/pluggable"
	_ "github.com/thanksloving/dynamic-plugin-server/repo"
	"github.com/thanksloving/dynamic-plugin-server/server"
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
