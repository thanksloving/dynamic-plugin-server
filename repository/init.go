package repository

import (
	"github.com/thanksloving/dynamic-plugin-server/pkg/pluggable"
)

func init() {
	err := pluggable.Register[*DemoParameter, *DemoResult]("SayHello", &Demo{})
	if err != nil {
		return
	}

}
