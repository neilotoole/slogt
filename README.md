# slogt

Bridge between Go `testing` and `golang.org/x/exp/slog` packages.

When tests execute, you want your `slog` output to be redirected
to the test's `*testing.T`, so that a test's log output is correlated
with that test. This package is a bridge between those worlds.

## Use

First, `go get` as per usual:

```shell
go get github.com/neilotoole/slogt
```

Then, use `slogt.New` to get a `*slog.Logger` that you can
use as normal.

```go
func TestSomething(t *testing.T) {
	log := slogt.New(t)

	// Use log as you normally would.
	log.Info("hello world")
}
```

## Deficiency

Calling `t.Log()` prints the callsite. However, given the available functionality
on `testing.T` (i.e. the `Helper` method), and the way `slog` is implemented,
there's no way to have the correct callsite printed.

There are a number of ways this could be fixed:

1. The Go team could implement a `testing.NewLogger(t)` function that effectively
   does what this package does, but it would have access to the `testing.T`'s
   internal state, and so could manipulate the calldepth.
2. The `testing.T` type could expose a `HelperN(depth int)` method that allows
   logging libraries and the like to manipulate the calldepth further.
3. The `slog` package could test if the handler implements an interface with
   method `Helper()`, and if so, invoke that method. This would need to be
   implemented in several spots in the codebase.
