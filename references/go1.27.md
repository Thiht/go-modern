# Go 1.27

Released 2026-08-19. Everything below appeared in this release: none of the
"after" blocks compile on 1.26. If your module's `go` directive is below 1.27,
close this file.

Sources: <https://go.dev/doc/go1.27> and the `api/go1.27.txt` file of the Go
repository. Entries are limited to changes that alter a decision you make while
writing code; toolchain, runtime and port changes are deliberately absent.

## Promoted fields as struct literal keys

> since 1.27 · mechanical · analyzer: embedlit

A struct literal key may now be a promoted field name, at any embedding depth.
The key must be the promoted name itself (`After`), not a selector chain
(`Cursor.After`), which still fails to compile.

**Before**

```go
package entry

type Cursor struct {
	After string
}

type Request struct {
	Cursor
	Limit int
}

func newRequest(after string, limit int) Request {
	return Request{
		Cursor: Cursor{After: after},
		Limit:  limit,
	}
}
```

**After**

```go
package entry

type Cursor struct {
	After string
}

type Request struct {
	Cursor
	Limit int
}

func newRequest(after string, limit int) Request {
	return Request{
		After: after,
		Limit: limit,
	}
}
```

## Generic methods

> since 1.27 · choice

A method declaration may declare its own type parameters, so a type no longer
needs one method per instantiation. Interface methods cannot declare type
parameters, and a generic method cannot implement one.

**Before**

```go
package entry

type Number interface {
	~int | ~float64
}

type Sums struct{}

func (Sums) Ints(xs []int) int {
	var total int
	for _, x := range xs {
		total += x
	}
	return total
}

func (Sums) Float64s(xs []float64) float64 {
	var total float64
	for _, x := range xs {
		total += x
	}
	return total
}
```

**After**

```go
package entry

type Number interface {
	~int | ~float64
}

type Sums struct{}

func (Sums) Sum[N Number](xs []N) N {
	var total N
	for _, x := range xs {
		total += x
	}
	return total
}
```

## Function type inference

> since 1.27 · choice

A generic function can be assigned to a variable of function type without
explicit type arguments; inference now happens in every assignment and
conversion context, including composite literals and channel sends.

**Before**

```go
package entry

import "slices"

func largest(xs []int) int {
	var f func([]int) int = func(s []int) int {
		return slices.Max(s)
	}
	return f(xs)
}
```

**After**

```go
package entry

import "slices"

func largest(xs []int) int {
	var f func([]int) int = slices.Max
	return f(xs)
}
```

## strings.CutLast

> since 1.27 · choice

`strings.CutLast` splits on the last occurrence and reports whether it found
one, replacing the `LastIndex` plus slicing dance.

**Before**

```go
package entry

import "strings"

func baseName(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
```

**After**

```go
package entry

import "strings"

func baseName(path string) string {
	if _, name, ok := strings.CutLast(path, "/"); ok {
		return name
	}
	return path
}
```

## bytes.CutLast

> since 1.27 · choice

The `[]byte` twin of `strings.CutLast`. Use it instead of `LastIndex` plus
reslicing, which is where off-by-one bugs live.

**Before**

```go
package entry

import "bytes"

func extension(name []byte) []byte {
	if i := bytes.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return nil
}
```

**After**

```go
package entry

import "bytes"

func extension(name []byte) []byte {
	if _, ext, ok := bytes.CutLast(name, []byte(".")); ok {
		return ext
	}
	return nil
}
```

## encoding/json/v2

> since 1.27 · choice

The new `encoding/json/v2` package takes options per call: strict UTF-8,
duplicate-name rejection and deterministic map ordering are available without
wrapping. `encoding/json` keeps its API and is now backed by the v2
implementation; migration is optional, but new code should start here.

**Before**

```go
package entry

import "encoding/json"

func encode(v any) ([]byte, error) {
	return json.Marshal(v)
}
```

**After**

```go
package entry

import "encoding/json/v2"

func encode(v any) ([]byte, error) {
	return json.Marshal(v, json.Deterministic(true))
}
```

## encoding/json/jsontext

> since 1.27 · choice

`jsontext` exposes the syntactic layer: tokens and values instead of typed
structs. Reach for it when you stream a large document, or when you need to
validate or rewrite JSON without materialising it.

**Before**

```go
package entry

import (
	"encoding/json"
	"io"
)

func writeNames(w io.Writer, names []string) error {
	payload, err := json.Marshal(names)
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}
```

**After**

```go
package entry

import (
	"encoding/json/jsontext"
	"io"
)

func writeNames(w io.Writer, names []string) error {
	enc := jsontext.NewEncoder(w)
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, name := range names {
		if err := enc.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}
```

## uuid

> since 1.27 · choice

The standard library now generates and parses UUIDs, so `github.com/google/uuid`
is no longer needed for the common cases. Prefer `NewV7` for identifiers that
land in a database: it is time-ordered, so inserts stay sequential. Keep the
dependency only if you need v1, v3 or v5.

**Before**

```go
package entry

import "github.com/google/uuid"

func newID() string {
	return uuid.NewString()
}
```

**After**

```go
package entry

import "uuid"

func newID() string {
	return uuid.NewV7().String()
}
```

## crypto/mldsa

> since 1.27 · choice

Post-quantum ML-DSA signatures (FIPS 204) are in the standard library, and
`crypto/x509` and `crypto/tls` understand the keys. Pick `MLDSA65` unless you
have an interoperability reason to go smaller or larger.

**Before**

```go
package entry

import "crypto/ed25519"

func newKey() (ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	return priv, err
}
```

**After**

```go
package entry

import "crypto/mldsa"

func newKey() (*mldsa.PrivateKey, error) {
	return mldsa.GenerateKey(mldsa.MLDSA65())
}
```

## crypto/tls MLKEM1024

> since 1.27 · choice

`tls.MLKEM1024` joins the post-quantum key exchanges. Listing it after
`X25519MLKEM768` negotiates the hybrid first and falls back to the larger
parameter set. Note that `tls.Config.Rand` is deprecated in this release: tests
that need determinism should call `testing/cryptotest.SetGlobalRandom`.

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
		CurvePreferences: []tls.CurveID{
			tls.X25519MLKEM768,
			tls.MLKEM1024,
		},
	}
}
```

## testing/synctest Sleep

> since 1.27 · choice

Inside a `synctest` bubble, `synctest.Sleep` advances the fake clock instead of
blocking a real goroutine, so a test that waits a minute runs in microseconds.
Use it in place of `time.Sleep` in any code the bubble drives.

**Before**

```go
package entry

import (
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	time.Sleep(2 * time.Second)
}
```

**After**

```go
package entry

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		synctest.Sleep(2 * time.Second)
	})
}
```

## httptest.NewTestServer

> since 1.27 · choice

`httptest.NewTestServer` builds an in-memory server that works inside a
`synctest` bubble: no listener, no real sockets, no port to pick. Pair it with
`synctest.Test` for handler tests that involve concurrency and time.

**Before**

```go
package entry

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(handle))
	defer srv.Close()
	if _, err := srv.Client().Get(srv.URL); err != nil {
		t.Fatal(err)
	}
}

func handle(http.ResponseWriter, *http.Request) {}
```

**After**

```go
package entry

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
)

func TestHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := httptest.NewTestServer(t, http.HandlerFunc(handle))
		defer srv.Close()
		if _, err := srv.Client().Get("http://test/"); err != nil {
			t.Fatal(err)
		}
	})
}

func handle(http.ResponseWriter, *http.Request) {}
```

## net/url Clone

> since 1.27 · choice

`(*url.URL).Clone` and `(url.Values).Clone` copy deeply. The `v := *u` idiom
shares the `User` pointer between the two values, so mutating credentials on the
copy rewrites the original: a real bug, not a style preference.

**Before**

```go
package entry

import "net/url"

func withScheme(u *url.URL, scheme string) *url.URL {
	v := *u
	v.Scheme = scheme
	return &v
}
```

**After**

```go
package entry

import "net/url"

func withScheme(u *url.URL, scheme string) *url.URL {
	v := u.Clone()
	v.Scheme = scheme
	return v
}
```
