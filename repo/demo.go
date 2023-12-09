package repo

import (
	"context"
	"fmt"

	"github.com/thanksloving/dynamic-plugin-server/pluggable"
)

var _ pluggable.Pluggable[*DemoParameter, *DemoResult] = &Demo{}

type (
	//@pluggable(qps=10 namespace=Default timeout=100ms)
	Demo struct {
	}
	DemoParameter struct {
		Name string `json:"name,omitempty" name:"name" desc:"姓名"`
	}

	DemoResult struct {
		Message string `json:"message,omitempty" name:"message"`
	}
)

func (d *Demo) Execute(_ context.Context, param *DemoParameter) (*DemoResult, error) {
	return &DemoResult{
		Message: fmt.Sprintf("hello %s", param.Name),
	}, nil
}
