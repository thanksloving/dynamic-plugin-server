# dynamic-plugin-server
A dynamic gRPC server without code generation. Write a plugin, register it, and the client can invoke it.

1. Wrote a plugin, implement Pluggable interface
```
var _ pluggable.Pluggable[*DemoParameter, *DemoResult] = &Demo{}

type (
	Demo struct {}

	DemoParameter struct {
		Name string `json:"name,omitempty" name:"name" desc:"姓名"`
	}

	DemoResult struct {
		Message string `json:"message,omitempty" name:"message"`
	}
)

func (d *Demo) Execute(_ context.Context, param *DemoParameter) (*DemoResult, error) {
    // TODO your plugin business
	return &DemoResult{
		Message: fmt.Sprintf("hello %s", param.Name),
	}, nil
}

```

2. Register the plugin.
```
err := pluggable.Register[*DemoParameter, *DemoResult]("SayHello", &Demo{})
```

3. Start the gRPC server.
```
dynamicService := server.NewDynamicService(pluggable.GetServiceDescriptors())
lis, err := net.Listen("tcp", ":52051")
if err != nil {
	panic(err)
}
log.Infof("server listening at %v", lis.Addr())
if e := dynamicService.Start(lis); e != nil {
	panic(e)
}
```

4. Then the client can get all the plugin metainfo, and invoke any plugin :)
```
conn, err := grpc.DialContext(context.Background(), ":52051",
	grpc.WithTransportCredentials(insecure.NewCredentials()),
)
stub := client.NewPluginStub(conn)

// get plugin meta info, it will auto invoke when new plugin stub
resp, err := ps.GetPluginMetaList(context.Background(), &pb.MetaRequest{})

// call the plugin server by meta info
result, err := stub.Call(context.Background(), pluggable.DefaultNamespace, "SayHello", map[string]any{"name": "plugin"})
```

### TODO
- [ ] meta info auto-generate support
- [x] meta info service
- [ ] parse meta info from the plugin
- [ ] version control
- [ ] benchmark