# errgroup

Provides a panic handling wrapper around [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) to avoid application crashes when a goroutine encounters a panic. In the event of a panic this wrapper will convert the panic into an error that is returned from the `Wait` function. In all other ways this wrapper behaves exactly as the wrapped `Group`.
