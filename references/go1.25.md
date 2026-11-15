# Go 1.25

Released 2025-08-19. Go 1.25 changed no language semantics, so every entry here
is library work: none of the "after" blocks compile on 1.24.

Sources: <https://go.dev/doc/go1.25> and the `api/go1.25.txt` file of the Go
repository. Entries are limited to changes that alter a decision you make while
writing code; toolchain, runtime and port changes are deliberately absent.

## sync.WaitGroup.Go

> since 1.25 · mechanical · analyzer: waitgroupgo

`WaitGroup.Go` runs the function in a new goroutine and does the `Add` and `Done`
itself, so a `defer` that someone will eventually forget disappears.

**Before**

```go
package entry

import "sync"

func run(fs []func()) {
	var wg sync.WaitGroup
	for _, f := range fs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}
```

**After**

```go
package entry

import "sync"

func run(fs []func()) {
	var wg sync.WaitGroup
	for _, f := range fs {
		wg.Go(func() {
			f()
		})
	}
	wg.Wait()
}
```

## testing/synctest

> since 1.25 · choice

A `synctest` bubble gives the test a fake clock: sleeps return instantly, and
`synctest.Wait` blocks until every goroutine in the bubble is durably blocked.
A test that used to wait for real time now runs in microseconds, and stops being
flaky on a loaded machine.

**Before**

```go
package entry

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	start := time.Now()
	time.Sleep(time.Hour)
	if time.Since(start) < time.Hour {
		t.Fatal("slept too little")
	}
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

func TestBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		time.Sleep(time.Hour)
		if time.Since(start) != time.Hour {
			t.Fatal("the fake clock did not advance")
		}
	})
}
```

## http.CrossOriginProtection

> since 1.25 · choice

`CrossOriginProtection` rejects cross-origin browser requests using the
`Sec-Fetch-Site` header and the `Origin`/`Host` comparison. It needs no token,
no session and no per-route configuration, so hand-written CSRF middleware has
no reason to exist any more.

**Before**

```go
package entry

import "net/http"

func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" && r.Method != http.MethodGet {
			http.Error(w, "cross-origin request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

**After**

```go
package entry

import "net/http"

func csrf(next http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(next)
}
```

## reflect.TypeAssert

> since 1.25 · choice

`reflect.TypeAssert[T]` is `v.Interface().(T)` without the allocation that
`Interface` costs. It is the only way to get a typed value out of a `reflect.Value`
in a hot path.

**Before**

```go
package entry

import "reflect"

func asString(v reflect.Value) (string, bool) {
	s, ok := v.Interface().(string)
	return s, ok
}
```

**After**

```go
package entry

import "reflect"

func asString(v reflect.Value) (string, bool) {
	return reflect.TypeAssert[string](v)
}
```

## os.Root WriteFile and MkdirAll

> since 1.25 · choice

`os.Root` gained the whole file API it was missing: `ReadFile`, `WriteFile`,
`MkdirAll`, `RemoveAll`, `Rename`, `Symlink` and friends. Every path operation
inside a directory can now stay confined to it.

**Before**

```go
package entry

import (
	"os"
	"path/filepath"
)

func writeFile(dir, name string, data []byte) error {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

**After**

```go
package entry

import (
	"os"
	"path/filepath"
)

func writeFile(dir, name string, data []byte) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()

	if err := root.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return root.WriteFile(name, data, 0o644)
}
```

## runtime/trace FlightRecorder

> since 1.25 · choice

A flight recorder keeps the last few seconds of execution trace in a ring buffer
and writes it out on demand, so a rare event can be captured in production
without running a trace for the whole process lifetime.

**Before**

```go
package entry

import (
	"io"
	"runtime/trace"
)

func snapshot(w io.Writer) error {
	// A full trace has to be started before the event and stopped after it.
	if err := trace.Start(w); err != nil {
		return err
	}
	defer trace.Stop()
	return nil
}
```

**After**

```go
package entry

import (
	"io"
	"runtime/trace"
	"time"
)

func snapshot(w io.Writer) error {
	rec := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:  10 * time.Second,
		MaxBytes: 1 << 20,
	})
	if err := rec.Start(); err != nil {
		return err
	}
	defer rec.Stop()

	_, err := rec.WriteTo(w)
	return err
}
```

## testing.T.Output

> since 1.25 · choice

`T.Output` returns a writer that lands in the test's own output stream, indented
like `t.Log`. Helpers stop taking a writer parameter or formatting strings they
would rather stream.

**Before**

```go
package entry

import "testing"

func logLines(t *testing.T, lines []string) {
	for _, line := range lines {
		t.Log(line)
	}
}
```

**After**

```go
package entry

import (
	"fmt"
	"testing"
)

func logLines(t *testing.T, lines []string) {
	w := t.Output()
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}
```

## hash.Cloner

> since 1.25 · choice

Every standard library hash now implements `hash.Cloner`, so a hash state can be
copied without the `BinaryMarshaler` round trip that only the concrete types
supported.

**Before**

```go
package entry

import (
	"crypto/sha256"
	"encoding"
	"hash"
)

func cloneHash(h hash.Hash) hash.Hash {
	state, err := h.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		panic(err)
	}
	next := sha256.New()
	if err := next.(encoding.BinaryUnmarshaler).UnmarshalBinary(state); err != nil {
		panic(err)
	}
	return next
}
```

**After**

```go
package entry

import "hash"

func cloneHash(h hash.Cloner) (hash.Cloner, error) {
	return h.Clone()
}
```
