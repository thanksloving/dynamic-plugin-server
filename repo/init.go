package repo

import "github.com/thanksloving/dynamic-plugin-server/pluggable"

func init() {
	defer pluggable.GetServiceDescriptors()
	err := pluggable.Register[*DemoParameter, *DemoResult]("SayHello", &Demo{})
	if err != nil {
		return
	}

}
