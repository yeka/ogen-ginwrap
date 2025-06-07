# ogen-ginwrap

This is a [gin](https://github.com/gin-gonic/gin) wrapper for [ogen-go](https://github.com/ogen-go/ogen), that works nicely with OpenTelemetry.

[ogen-go](https://github.com/ogen-go/ogen) is an OpenAPI generator that generate stric go code.

You can use the `ogen-go` generated handler directly as http.Handler, or wrap the generated handler in gin like so:
```go
server := gen.NewServer(MyHandler{}) // generated strict interface

r := gin.Default()
r.Any("/*any", gin.WrapH(server))
```

The problem is, when using OpenTelemetry traces, all path become `/*any`, which isn't really helpful for observing your system.

This wrapper generates all path from OpenAPI document into coresponding gin routes and make your observability happy.

How to generate:
```bash
go install github.com/yeka/ogen-ginwrap@latest
./ogen-ginwrap -file openapi.yaml -pkg ginwrap -out generated/ginwrap/ginwrap.go
```

From your project:
```go
server := gen.NewServer(MyHandler{}) // generated strict interface

r := gin.Default()
r.Use(otelgin.Middleware("your-service"))
ginwrap.RegisterRoutes(r, server, "")
```

Adding prefix to ogen & gin:
```go
server := gen.NewServer(MyHandler{}, gen.WithPathPrefix("/v1")) // generated strict interface

r := gin.Default()
r.Use(otelgin.Middleware("your-service"))
ginwrap.RegisterRoutes(r, server, "/v1")
```