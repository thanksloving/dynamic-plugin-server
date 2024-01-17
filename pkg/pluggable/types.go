package pluggable

import (
	"context"
)

type (
	// Pluggable is plugin, it should be implemented
	Pluggable[I, O any] interface {
		Execute(ctx context.Context, param I) (O, error)
	}

	// CustomCacheKey is used to generate custom cache key for plugin parameters
	CustomCacheKey interface {
		GenerateKey(namespace, pluginName string) string
	}

	Input struct {
		Item
		Optional bool
		// list the options of the value if the value is limited
		Options []any
	}

	Output struct {
		Item
	}

	Item struct {
		Name string
		Type string
		Desc string
	}
)
