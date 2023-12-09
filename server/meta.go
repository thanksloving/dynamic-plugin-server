package server

import (
	"context"

	"github.com/pkg/errors"
)

// Meta TODO add version: local timestamp
func (ds *dynamicService) Meta(_ context.Context, methodName string) (interface{}, error) {
	switch methodName {
	default:
		return nil, errors.Errorf("unknown method %s", methodName)
	}
}
