package flagd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr/funcr"
	of "github.com/open-feature/go-sdk/openfeature"
)

const loggerTestFlags = `{"flags":{"bool":{"state":"ENABLED","variants":{"on":true,"off":false},"defaultVariant":"on"}}}`

// swapStderr points os.Stderr at a pipe and returns a func that restores it and returns
// everything written to the pipe. It must run before NewProvider, because the default zap
// logger binds os.Stderr when it is constructed.
func swapStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w

	out := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		out <- string(b)
	}()

	var once sync.Once
	var got string
	stop := func() string {
		once.Do(func() {
			os.Stderr = orig
			_ = w.Close()
			got = <-out
			_ = r.Close()
		})
		return got
	}
	t.Cleanup(func() { stop() })
	return stop
}

func TestInProcessResolversRespectWithLogger(t *testing.T) {
	tests := []struct {
		name       string
		opts       []ProviderOption
		withLogger bool
	}{
		{name: "in-process with offline file", opts: []ProviderOption{WithInProcessResolver()}, withLogger: true},
		{name: "file resolver", opts: []ProviderOption{WithFileResolver()}, withLogger: true},
		{name: "in-process without WithLogger", opts: []ProviderOption{WithInProcessResolver()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "flags.json")
			if err := os.WriteFile(path, []byte(loggerTestFlags), 0o600); err != nil {
				t.Fatal(err)
			}

			var mu sync.Mutex
			var captured []string
			capture := funcr.New(func(prefix, args string) {
				mu.Lock()
				defer mu.Unlock()
				captured = append(captured, args)
			}, funcr.Options{Verbosity: 10})
			capturedContains := func(s string) bool {
				mu.Lock()
				defer mu.Unlock()
				for _, line := range captured {
					if strings.Contains(line, s) {
						return true
					}
				}
				return false
			}

			stopStderr := swapStderr(t)

			opts := append(tt.opts, WithOfflineFilePath(path), WithOfflinePollMs(50))
			if tt.withLogger {
				opts = append(opts, WithLogger(capture))
			}
			p, err := NewProvider(opts...)
			if err != nil {
				t.Fatal(err)
			}
			if err := p.Init(of.EvaluationContext{}); err != nil {
				t.Fatal(err)
			}

			// the poll watcher only reports a change when the modtime moves forward
			if err := os.WriteFile(path, []byte(loggerTestFlags), 0o600); err != nil {
				t.Fatal(err)
			}
			future := time.Now().Add(time.Hour)
			if err := os.Chtimes(path, future, future); err != nil {
				t.Fatal(err)
			}

			if tt.withLogger {
				deadline := time.Now().Add(5 * time.Second)
				for !capturedContains("filepath event") && time.Now().Before(deadline) {
					time.Sleep(20 * time.Millisecond)
				}
			}
			p.Shutdown()
			stderr := stopStderr()

			if !tt.withLogger {
				if !strings.Contains(stderr, `"level":"info"`) {
					t.Fatalf("expected default zap JSON on stderr, got %q", stderr)
				}
				return
			}
			if !capturedContains("filepath event") {
				mu.Lock()
				defer mu.Unlock()
				t.Fatalf("expected a filepath event line in the WithLogger logger, got %q", captured)
			}
			if stderr != "" {
				t.Fatalf("expected nothing on stderr when WithLogger is set, got %q", stderr)
			}
		})
	}
}
