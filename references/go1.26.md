# Go 1.26

Released 2026-02-10. Everything below appeared in this release: none of the
"after" blocks compile on 1.25.

Sources: <https://go.dev/doc/go1.26> and the `api/go1.26.txt` file of the Go
repository. Entries are limited to changes that alter a decision you make while
writing code; toolchain, runtime and port changes are deliberately absent, which
leaves out the garbage collector and the experimental packages.

## new(expr)

> since 1.26 · choice

`new` accepts a value instead of a type, so an optional field can be populated in
one expression. `go fix` also rewrites the `func f(x T) *T { return &x }` helper
into `return new(x)` and inlines its call sites, which usually makes the helper
dead.

**Before**

```go
package entry

type Person struct {
	Name string
	Age  *int
}

func person(name string, age int) Person {
	return Person{Name: name, Age: ptr(age)}
}

func ptr(v int) *int {
	p := new(int)
	*p = v
	return p
}
```

**After**

```go
package entry

type Person struct {
	Name string
	Age  *int
}

func person(name string, age int) Person {
	return Person{Name: name, Age: new(age)}
}
```

## errors.AsType

> since 1.26 · mechanical · analyzer: errorsastype

`errors.AsType[T]` is typed, allocates nothing, and keeps the matched error in
scope, so the `var target *T` declaration before the `if` disappears.

**Before**

```go
package entry

import "errors"

type notFound struct{}

func (notFound) Error() string { return "not found" }

func report(err error) string {
	var target *notFound
	if errors.As(err, &target) {
		return target.Error()
	}
	return ""
}
```

**After**

```go
package entry

import "errors"

type notFound struct{}

func (notFound) Error() string { return "not found" }

func report(err error) string {
	if target, ok := errors.AsType[*notFound](err); ok {
		return target.Error()
	}
	return ""
}
```

## Recursive generic types

> since 1.26 · choice

A generic type may refer to itself in its own constraint list, which removes the
second type parameter that used to carry the relationship explicitly.

**Before**

```go
package entry

type Adder[A any] interface {
	Add(A) A
}

func sum[A any, T Adder[A]](xs []T, zero A) A {
	total := zero
	for _, x := range xs {
		total = x.Add(total)
	}
	return total
}
```

**After**

```go
package entry

type Adder[A Adder[A]] interface {
	Add(A) A
}

func sum[A Adder[A]](xs []A, zero A) A {
	total := zero
	for _, x := range xs {
		total = x.Add(total)
	}
	return total
}
```

## crypto/hpke

> since 1.26 · choice

HPKE (RFC 9180) is in the standard library, including post-quantum hybrid KEMs
such as `MLKEM768X25519`. The one-shot `Seal` and `Open` cover the common cases;
`Sender` and `Recipient` expose the exported-secret API.

**Before**

```go
package entry

import "golang.org/x/crypto/hpke"

func seal(pk hpke.PublicKey, msg []byte) ([]byte, error) {
	return hpke.Seal(pk, hpke.HKDFSHA256(), hpke.AES128GCM(), []byte("ctx"), msg)
}
```

**After**

```go
package entry

import "crypto/hpke"

func seal(pk hpke.PublicKey, msg []byte) ([]byte, error) {
	return hpke.Seal(pk, hpke.HKDFSHA256(), hpke.AES128GCM(), []byte("ctx"), msg)
}
```

## slog.NewMultiHandler

> since 1.26 · choice

`NewMultiHandler` fans a record out to several handlers, so the hand-written
handler that forwards `Enabled`, `Handle`, `WithAttrs` and `WithGroup` can go.

**Before**

```go
package entry

import (
	"context"
	"log/slog"
)

type multi struct{ handlers []slog.Handler }

func (m multi) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m multi) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (m multi) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return multi{next}
}

func (m multi) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithGroup(name)
	}
	return multi{next}
}
```

**After**

```go
package entry

import "log/slog"

func logger(hs ...slog.Handler) *slog.Logger {
	return slog.New(slog.NewMultiHandler(hs...))
}
```

## reflect iterators

> since 1.26 · choice

`reflect.Type` and `reflect.Value` expose fields, methods and signature parameters
as iterators, so the `NumField` plus `Field(i)` loop has a replacement that reads
like the rest of the range-over-func code.

**Before**

```go
package entry

import "reflect"

func fieldNames(t reflect.Type) []string {
	var names []string
	for i := 0; i < t.NumField(); i++ {
		names = append(names, t.Field(i).Name)
	}
	return names
}
```

**After**

```go
package entry

import "reflect"

func fieldNames(t reflect.Type) []string {
	var names []string
	for f := range t.Fields() {
		names = append(names, f.Name)
	}
	return names
}
```

## testing.T.ArtifactDir

> since 1.26 · choice

Artifacts produced by a test used to land in `t.TempDir`, which is deleted when
the test ends. `t.ArtifactDir` returns a directory kept when `go test -artifacts`
is passed, and a temporary one otherwise.

**Before**

```go
package entry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "graph.png"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
```

**After**

```go
package entry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender(t *testing.T) {
	if err := os.WriteFile(filepath.Join(t.ArtifactDir(), "graph.png"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
```

## net.Dialer typed dials

> since 1.26 · choice

`Dialer` gained `DialTCP`, `DialUDP`, `DialIP` and `DialUnix`, which take
`netip.AddrPort` and return the concrete connection type. The `net.Dial` plus type
assertion pair disappears, and so does its panic.

**Before**

```go
package entry

import (
	"context"
	"net"
)

func dial(ctx context.Context, addr string) (*net.TCPConn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
```

**After**

```go
package entry

import (
	"context"
	"net"
	"net/netip"
)

func dial(ctx context.Context, raddr netip.AddrPort) (*net.TCPConn, error) {
	var d net.Dialer
	return d.DialTCP(ctx, "tcp", netip.AddrPort{}, raddr)
}
```

## httputil ReverseProxy.Rewrite

> since 1.26 · choice

`ReverseProxy.Director` is deprecated: a client can strip headers a `Director`
added by marking them hop-by-hop. `Rewrite` receives the inbound and the outbound
request, which is why it can sanitise the hop-by-hop headers first.

**Before**

```go
package entry

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func proxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
		},
	}
}
```

**After**

```go
package entry

import (
	"net/http/httputil"
	"net/url"
)

func proxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
		},
	}
}
```

## testing/cryptotest.SetGlobalRandom

> since 1.26 · choice

The `random` parameter of the `crypto/...` key generation and signing functions is
now ignored, so a test that injected a deterministic reader silently lost its
determinism. The replacement is `cryptotest.SetGlobalRandom`.

**Before**

```go
package entry

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_ = key
}
```

**After**

```go
package entry

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"testing"
	"testing/cryptotest"
)

func TestKey(t *testing.T) {
	cryptotest.SetGlobalRandom(t, 42)

	key, err := ecdsa.GenerateKey(elliptic.P256(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = key
}
```

## bytes.Buffer.Peek

> since 1.26 · choice

`Peek` reads the next n bytes without advancing the buffer and reports `io.EOF`
when there are fewer, replacing the `b.Bytes()[:n]` slice arithmetic that panics
on a short buffer.

**Before**

```go
package entry

import "bytes"

func peek(b *bytes.Buffer, n int) []byte {
	data := b.Bytes()
	if len(data) < n {
		return data
	}
	return data[:n]
}
```

**After**

```go
package entry

import "bytes"

func peek(b *bytes.Buffer, n int) ([]byte, error) {
	return b.Peek(n)
}
```
