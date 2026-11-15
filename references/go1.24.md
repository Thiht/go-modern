# Go 1.24

Released 2025-02-11. Everything below appeared in this release: none of the
"after" blocks compile on 1.23.

Sources: <https://go.dev/doc/go1.24> and the `api/go1.24.txt` file of the Go
repository. Entries are limited to changes that alter a decision you make while
writing code; toolchain, runtime and port changes are deliberately absent.

## Generic type aliases

> since 1.24 · choice

An alias may now carry type parameters, so a concrete instantiation is the same
type as its generic parent. A defined type is not: assignment between the two
needs a conversion, and methods do not carry over.

**Before**

```go
package entry

type Set[T comparable] map[T]struct{}

// A copy of the generic type, not the same type.
type IntSet map[int]struct{}
```

**After**

```go
package entry

type Set[T comparable] map[T]struct{}

// IntSet is Set[int]: a Set[int] can be passed wherever an IntSet is expected.
type IntSet = Set[int]
```

## os.Root

> since 1.24 · choice

`os.Root` confines every operation to a directory, so a name containing `..` or
an absolute path cannot escape it. `filepath.Join` plus `os.Open` does not: it
resolves the traversal before the open.

**Before**

```go
package entry

import (
	"os"
	"path/filepath"
)

func readConfig(dir, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, name))
}
```

**After**

```go
package entry

import (
	"io"
	"os"
)

func readConfig(dir, name string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}
```

## weak

> since 1.24 · choice

`weak.Pointer` references an object without keeping it alive, which is what a
cache needs: an entry nobody holds any more should be collectable, and a stale
entry should read as nil instead of pinning memory.

**Before**

```go
package entry

import "sync"

type Entry struct{ Payload []byte }

// Cache keeps every entry alive, even the ones nobody uses any more.
type Cache struct {
	mu    sync.Mutex
	items map[string]*Entry
}

func (c *Cache) Put(key string, e *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = make(map[string]*Entry)
	}
	c.items[key] = e
}
```

**After**

```go
package entry

import (
	"sync"
	"weak"
)

type Entry struct{ Payload []byte }

// Cache lets the collector reclaim entries nobody holds any more.
type Cache struct {
	mu    sync.Mutex
	items map[string]weak.Pointer[Entry]
}

func (c *Cache) Put(key string, e *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = make(map[string]weak.Pointer[Entry])
	}
	c.items[key] = weak.Make(e)
}
```

## runtime.AddCleanup

> since 1.24 · choice

`AddCleanup` attaches a release function to a value without the ordering and
resurrection problems of `SetFinalizer`, and several cleanups may share one
object. The argument must be the resource, never the object itself, or the
cleanup keeps it alive forever.

**Before**

```go
package entry

import "runtime"

type Pool struct{ id uint64 }

func newPool(id uint64) *Pool {
	p := &Pool{id: id}
	runtime.SetFinalizer(p, func(p *Pool) { release(p.id) })
	return p
}

func release(id uint64) {}
```

**After**

```go
package entry

import "runtime"

type Pool struct{ id uint64 }

func newPool(id uint64) *Pool {
	p := &Pool{id: id}
	runtime.AddCleanup(p, release, id)
	return p
}

func release(id uint64) {}
```

## strings.SplitSeq

> since 1.24 · mechanical · analyzer: stringsseq

Ranging over `strings.SplitSeq` yields the same substrings as `strings.Split`
without materialising the slice, which matters on a hot path.

**Before**

```go
package entry

import "strings"

func parts(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		out = append(out, p)
	}
	return out
}
```

**After**

```go
package entry

import "strings"

func parts(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		out = append(out, p)
	}
	return out
}
```

## t.Context

> since 1.24 · choice

`t.Context` returns a context cancelled just before the test's cleanup functions
run, so a test never has to build and defer its own `WithCancel`.

**Before**

```go
package entry

import (
	"context"
	"testing"
)

func TestFetch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fetch(ctx); err != nil {
		t.Fatal(err)
	}
}

func fetch(context.Context) error { return nil }
```

**After**

```go
package entry

import (
	"context"
	"testing"
)

func TestFetch(t *testing.T) {
	if err := fetch(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func fetch(context.Context) error { return nil }
```

## b.Loop

> since 1.24 · choice

`b.Loop` replaces the `b.N` loop: it keeps the body from being optimised away,
resets the timer, and never runs the body a zero number of times.

**Before**

```go
package entry

import "testing"

func BenchmarkSum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sum([]int{1, 2, 3})
	}
}

func sum(xs []int) int {
	var total int
	for _, x := range xs {
		total += x
	}
	return total
}
```

**After**

```go
package entry

import "testing"

func BenchmarkSum(b *testing.B) {
	for b.Loop() {
		sum([]int{1, 2, 3})
	}
}

func sum(xs []int) int {
	var total int
	for _, x := range xs {
		total += x
	}
	return total
}
```

## crypto/rand.Text

> since 1.24 · choice

`rand.Text` returns a random string with at least 128 bits of entropy, in the
RFC 4648 base32 alphabet. It replaces the read, encode and error-check dance
every service writes for tokens.

**Before**

```go
package entry

import (
	"crypto/rand"
	"encoding/base64"
)

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
```

**After**

```go
package entry

import "crypto/rand"

func newToken() string {
	return rand.Text()
}
```

## crypto/sha3

> since 1.24 · choice

SHA-3 and SHAKE moved into the standard library. Drop the `golang.org/x/crypto`
dependency: the import path is the only thing that changes.

**Before**

```go
package entry

import "golang.org/x/crypto/sha3"

func digest(b []byte) []byte {
	sum := sha3.Sum256(b)
	return sum[:]
}
```

**After**

```go
package entry

import "crypto/sha3"

func digest(b []byte) []byte {
	sum := sha3.Sum256(b)
	return sum[:]
}
```

## crypto/hkdf

> since 1.24 · choice

HKDF is in the standard library, with a one-shot `Key` that takes the info string
directly instead of wrapping a reader. `crypto/pbkdf2` arrived in the same
release for password hashing.

**Before**

```go
package entry

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

func derive(secret, salt []byte) ([]byte, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(hkdf.New(sha256.New, secret, salt, []byte("session")), key)
	return key, err
}
```

**After**

```go
package entry

import (
	"crypto/hkdf"
	"crypto/sha256"
)

func derive(secret, salt []byte) ([]byte, error) {
	return hkdf.Key(sha256.New, secret, salt, "session", 32)
}
```

## crypto/mlkem

> since 1.24 · choice

ML-KEM (FIPS 203) is in the standard library. ML-KEM-768 is the parameter set to
start with; the decapsulation key carries its encapsulation key.

**Before**

```go
package entry

import "golang.org/x/crypto/mlkem"

func newKey() (*mlkem.DecapsulationKey768, error) {
	return mlkem.GenerateKey768()
}
```

**After**

```go
package entry

import "crypto/mlkem"

func newKey() (*mlkem.DecapsulationKey768, error) {
	return mlkem.GenerateKey768()
}
```

## crypto/tls X25519MLKEM768

> since 1.24 · choice

The post-quantum hybrid `X25519MLKEM768` is negotiated by default, so an explicit
`CurvePreferences` list written before this release silently downgrades the
handshake. Only pin curves when you must interoperate with a peer that cannot
negotiate the hybrid.

**Before**

```go
package entry

import "crypto/tls"

func config() *tls.Config {
	return &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}
}
```

**After**

```go
package entry

import "crypto/tls"

func config() *tls.Config {
	return &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519MLKEM768, tls.X25519},
	}
}
```

## net/http Protocols

> since 1.24 · choice

`http.Server.Protocols` configures HTTP/1, HTTP/2 and unencrypted HTTP/2 without
the `golang.org/x/net/http2` machinery, and the same field exists on
`http.Transport`.

**Before**

```go
package entry

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func newServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h2c.NewHandler(h, &http2.Server{})}
}
```

**After**

```go
package entry

import "net/http"

func newServer(h http.Handler) *http.Server {
	srv := &http.Server{Handler: h}
	srv.Protocols = new(http.Protocols)
	srv.Protocols.SetHTTP1(true)
	srv.Protocols.SetUnencryptedHTTP2(true)
	return srv
}
```

## go/types iterators

> since 1.24 · choice

The `go/types` accessors that took an index now also return iterators, so tools
stop writing `NumMethods` plus `Method(i)` loops. The indexed methods still
exist; the iterator is the idiomatic form.

**Before**

```go
package entry

import "go/types"

func methodNames(t *types.Interface) []string {
	var names []string
	for i := 0; i < t.NumMethods(); i++ {
		names = append(names, t.Method(i).Name())
	}
	return names
}
```

**After**

```go
package entry

import "go/types"

func methodNames(t *types.Interface) []string {
	var names []string
	for fn := range t.Methods() {
		names = append(names, fn.Name())
	}
	return names
}
```
