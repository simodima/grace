# Grace

A graceful shutdown HTTP server


```go
err := grace.RunGracefully(
    router,
    grace.WithBindAddress(":8080"),
    grace.WithShutdownTimeout(5*time.Second),
)
```