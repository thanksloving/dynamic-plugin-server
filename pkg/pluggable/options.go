package pluggable

import (
	"time"
)

// Desc is the description of plugin
func Desc(desc string) Option {
	return func(meta *PluginMeta) {
		meta.Desc = desc
	}
}

// QPS is the qps of plugin, default is unlimited
func QPS(limit int) Option {
	return func(meta *PluginMeta) {
		meta.QPS = &limit
	}
}

// Namespace is the namespace of plugin, default is "default"
func Namespace(namespace string) Option {
	return func(meta *PluginMeta) {
		meta.Namespace = namespace
	}
}

// Timeout is the timeout of plugin, default is 100ms
func Timeout(timeout time.Duration) Option {
	return func(meta *PluginMeta) {
		t := timeout.Milliseconds()
		meta.Timeout = &t
	}
}

// CacheTime is the cache time of plugin result
func CacheTime(ttl time.Duration) Option {
	return func(meta *PluginMeta) {
		t := ttl.Milliseconds()
		meta.CacheTime = &t
	}
}
