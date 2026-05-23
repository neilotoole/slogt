# slogt

[![Go](https://github.com/neilotoole/slogt/actions/workflows/go.yml/badge.svg)](https://github.com/neilotoole/slogt/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/neilotoole/slogt/graph/badge.svg)](https://codecov.io/gh/neilotoole/slogt)
[![Go Reference](https://pkg.go.dev/badge/github.com/neilotoole/slogt/v2.svg)](https://pkg.go.dev/github.com/neilotoole/slogt/v2)

`slogt` is a bridge between Go stdlib [`testing`](https://pkg.go.dev/testing) pkg
and [`log/slog`](https://pkg.go.dev/log/slog).


The problem: when tests execute, your `slog` output goes directly to `stdout`,
unlike a call to `t.Log`, which is correlated with your test's execution.

```go
func TestSlog_Ugly(t *testing.T) {
   log := slog.New(slog.NewTextHandler(os.Stdout, nil))
   t.Log("I am indented correctly")
   log.Info("But I am not")
}
```

Produces:

```text
=== RUN   TestSlog_Ugly
    slogt_test.go:21: I am indented correctly
time=2023-04-01T11:29:27.236-06:00 level=INFO msg="But I am not"
```

Note the second line (produced via `slog`).

`slogt` bridges those packages.

```go
func TestSlogt_Pretty(t *testing.T) {
	log := slogt.New(t)
	t.Log("I am indented correctly")
	log.Info("And so am I")
}
```

Produces:

```text
=== RUN   TestSlogt_Pretty
    slogt_test.go:27: I am indented correctly
    time=2026-05-23T10:00:00.000-06:00 level=INFO msg="And so am I"
```


## Usage

Run `go get` as per procedure:

```shell
go get -u github.com/neilotoole/slogt/v2
```

> [!NOTE]
> `slogt/v2` requires **Go 1.25 or later** (it routes output through
> [`testing.TB.Output`](https://pkg.go.dev/testing#T.Output)).

Then, use `slogt.New` to get a `*slog.Logger` that you can
use as you normally would.

```go
func TestText(t *testing.T) {
   log := slogt.New(t)
   log.Info("hello world")
}
```

Produces:

```text
=== RUN   TestText
    time=2026-05-23T10:00:00.000-06:00 level=INFO msg="hello world"
```

In practice, you would pass the `*slog.Logger` returned from `slogt.New` to
the component under test. For example:

```go
func TestApp(t *testing.T) {
  log := slogt.New(t)
  
  app := app.New(log, ...) // other dependencies

  result, err := app.DepositMoney(100)
  require.NoError(t, err)
  require.Equal(t, 100, result.Balance)
}
```

If the `app.DepositMoney` method logs anything, its output is routed to
`t.Output()` and correlated with the running test.

### Options

The default output is text, i.e. a `slog.TextHandler.` You can
specify JSON using the `slogt.JSON()` option.

```go
func TestJSON(t *testing.T) {
   log := slogt.New(t, slogt.JSON())
   log.Info("hello world")
}
```

Produces:

```text
=== RUN   TestJSON
    {"time":"2026-05-23T10:00:00.000-06:00","level":"INFO","msg":"hello world"}
```

To switch the default handler:

```go
func init() {
    slogt.SetDefault(slogt.JSON())
}
```

When `Text()` and `JSON()` aren't enough — a non-default level, `AddSource: true`,
a custom `ReplaceAttr`, or a third-party handler — reach for `slogt.Factory()`.
slogt calls your function with the test's writer (`t.Output()`), and you return
any `slog.Handler` that writes to it. (Since v2 removed the exported `Bridge`
type, `Factory` is _the_ way to plug in a custom handler.)

```go
func TestSomething(t *testing.T) {
    // This factory returns a slog.Handler using slog.LevelError.
    f := slogt.Factory(func(w io.Writer) slog.Handler {
       opts := &slog.HandlerOptions{
           Level: slog.LevelError,
       }
       return slog.NewTextHandler(w, opts)
    })

    log := slogt.New(t, f)
}
```

### Showing the caller

By default, log lines carry no callsite (writing through `t.Output()` adds none).
If you want the real caller, enable `AddSource: true` on the handler:

```go
func TestCaller(t *testing.T) {
	f := slogt.Factory(func(w io.Writer) slog.Handler {
		opts := &slog.HandlerOptions{
			AddSource: true,
		}

		return slog.NewTextHandler(w, opts)
	})

	log := slogt.New(t, f)
	log.Info("Show me the real callsite")
}
```

Produces (note the correct `source` attribute — the actual call site, not slog's
internals):

```text
=== RUN   TestCaller
    time=2026-05-23T10:00:00.000-06:00 level=INFO source=.../slogt_test.go:118 msg="Show me the real callsite"
```

## History

`slogt` originally had a notable deficiency: every log line was prefixed with a
**bogus callsite** pointing into `slog`'s internals (e.g. `logger.go:230`) rather
than the real `log.Info()` call site. This was unavoidable — the only sink
available was `t.Log(string)`, a function call rather than an `io.Writer`, so
`slogt` had to buffer slog's output and forward it through `t.Log`, which then
attributed each line to `slog`'s own internals (the frame that called `Handle`)
rather than the real call site. `t.Helper()` could not mark enough frames to fix it.

That limitation directly motivated the upstream Go proposal
[golang/go#59928](https://github.com/golang/go/issues/59928), which was accepted
and shipped in **Go 1.25** as
[`testing.TB.Output()`](https://pkg.go.dev/testing#T.Output): an `io.Writer` that
writes to the test log without prepending a source location. As of `v2.0.0`,
slogt writes straight to `t.Output()`, so the bogus prefix is gone — and the real
caller is available via `AddSource: true` (see [Showing the caller](#showing-the-caller)).

> [!NOTE]
> This deficiency is what motivated [golang/go#59928](https://github.com/golang/go/issues/59928).
> The original issue (by [@earthboundkid](https://github.com/earthboundkid)):
>
> > In a test, you often want to mock out the logger. It would be nice to be able to
> > call t.Slog() and get a log/slog logger that send output to t.Log() with the
> > correct caller information.
> >
> > See https://github.com/neilotoole/slogt for an example of a third party library
> > providing this functionality, but note that it cannot provide correct caller
> > information:
> >
> > > Alas, given the available functionality on testing.T (i.e. the Helper method),
> > > and how slog is implemented, there's no way to have the correct callsite printed.
> >
> > It seems like this needs to be done on the Go side to fix the callsite.

## Changelog

### v2.0.0 (2026-05-23)

- **Breaking:** require Go 1.25 and route output through
  [`testing.TB.Output()`](https://pkg.go.dev/testing#T.Output), eliminating the
  bogus slog-internal callsite prefix (see [History](#history)).
- **Breaking:** remove the exported `Bridge` type (an internal workaround that is
  no longer needed). The module path is now `github.com/neilotoole/slogt/v2`.
- **Breaking:** replace the mutable `Default` package var with the
  concurrency-safe `SetDefault` function.
- Drop the stale `golang.org/x/exp` dependency; test CI across Go 1.25.x and stable.

### v1.1.0 (2023-08-12)

- Require Go 1.21; switch from `golang.org/x/exp/slog` to the standard library
  `log/slog`.

### v1.0.1 (2023-05-31)

- Update README examples for the newer `golang.org/x/exp/slog` API.

### v1.0.0 (2023-04-03)

- Initial release: bridge between `testing.T` and `golang.org/x/exp/slog`.
