// Command control is the OpenFeature Provider TCK control server for a Flipt
// backend under test.
//
// It runs as PID 1 inside a container built on the ghcr.io/flipt-io/flipt:v2
// image and implements the backend control API described in the OpenFeature
// provider-tck control-api.yaml: it starts and stops the /flipt process inside
// the running container and seeds and mutates the canonical flag set through
// Flipt's v2 management API. Nothing ever stops the container itself, which is
// the no-container-restart invariant the control API requires.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	// controlAddr is the address the control API listens on inside the
	// container. The TCK discovers it through the docker-compose port mapping.
	controlAddr = ":8080"

	// fliptHTTPPort is the HTTP port /flipt server listens on inside the
	// container.
	fliptHTTPPort = "8888"
	fliptBaseURL  = "http://127.0.0.1:" + fliptHTTPPort

	// environmentKey and namespaceKey are the Flipt environment and namespace
	// the canonical flags are seeded into, matching the OpenFeature client's
	// defaults of "default".
	environmentKey = "default"
	namespaceKey   = "default"
)

// flagDef is the baseline definition of one canonical flag, expressed the way
// Flipt holds it. The baseline is translated from the embedded canonical flag
// set at boot (see loadBaseline in seed.go), never hardcoded here.
type flagDef struct {
	key     string
	payload flagPayload
}

// control is the running backend under test.
type control struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	flipt    *fliptAPI
	baseline []flagDef // canonical flag set, translated once at boot
	changing bool      // tracks which way /change last toggled changing-flag
}

func (c *control) seed(ctx context.Context) error {
	for _, seg := range segmentsFor() {
		payload, err := json.Marshal(seg)
		if err != nil {
			return fmt.Errorf("marshal segment %q: %w", seg.Key, err)
		}
		if err := c.flipt.putOrCreate(ctx, resourceRequest{
			NamespaceKey: namespaceKey,
			Key:          seg.Key,
			Payload:      payload,
		}); err != nil {
			return fmt.Errorf("seed segment %q: %w", seg.Key, err)
		}
	}

	for _, def := range c.baseline {
		fp := def.payload
		if fp.Key == "targeting-key-flag" {
			withRule(&fp, rule{
				SegmentOperator: segmentOperatorAll,
				Segments:        []string{"targetingsg"},
				Distributions:   []distribution{{Variant: "hit", Rollout: 100}},
			})
		}
		payload, err := json.Marshal(fp)
		if err != nil {
			return fmt.Errorf("marshal flag %q: %w", fp.Key, err)
		}
		if err := c.flipt.putOrCreate(ctx, resourceRequest{
			NamespaceKey: namespaceKey,
			Key:          fp.Key,
			Payload:      payload,
		}); err != nil {
			return fmt.Errorf("seed flag %q: %w", fp.Key, err)
		}
	}
	return nil
}

// startBoot spawns /flipt and seeds the canonical flag set, blocking until the
// new state is actually being served.
func (c *control) startBoot(ctx context.Context, config string) error {
	if config == "" {
		config = "default"
	}
	if config != "default" {
		return fmt.Errorf("unknown configuration %q: this testbed serves only %q", config, "default")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.spawnLocked(ctx); err != nil {
		return err
	}
	if err := c.seed(ctx); err != nil {
		return err
	}

	return nil
}

// spawnLocked makes sure the /flipt child is running. The Testcontainers
// compose wait requires both the control port and the flipt HTTP port to
// listen before the suite posts /start, so flipt must be up from container
// start; /start only (re)seeds once the child is alive. The lock must already
// be held.
func (c *control) spawnLocked(ctx context.Context) error {
	if c.cmd != nil {
		return nil
	}

	cmd := exec.Command("/flipt", "server")
	cmd.Env = append(os.Environ(), "FLIPT_SERVER_HTTP_PORT="+fliptHTTPPort)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start /flipt: %w", err)
	}
	c.cmd = cmd

	if err := c.flipt.awaitUp(ctx, 20*time.Second); err != nil {
		return err
	}
	return nil
}

// stopBoot kills the /flipt child process so the backend becomes unreachable
// while the container stays up. Stopping an already-stopped backend succeeds.
func (c *control) stopBoot(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.killLocked(ctx)
}

// killLocked SIGKILLs the flipt child's process group and waits for it to
// exit. The lock must already be held.
func (c *control) killLocked(ctx context.Context) error {
	if c.cmd == nil {
		return nil
	}
	cmd := c.cmd
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	select {
	case <-waitCh:
	case <-time.After(10 * time.Second):
		return fmt.Errorf("flipt did not exit after SIGKILL")
	case <-ctx.Done():
		return ctx.Err()
	}
	c.cmd = nil
	return nil
}

// changeFlag toggles changing-flag's default variant between the two variant
// keys the baseline holds for it. The mutation persists until the next /start
// or /reset, as the control API requires.
func (c *control) changeFlag(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changing = !c.changing
	var base *flagPayload
	for i := range c.baseline {
		if c.baseline[i].key == "changing-flag" {
			base = &c.baseline[i].payload
			break
		}
	}
	if base == nil {
		return fmt.Errorf("changing-flag not in baseline")
	}
	if len(base.Variants) != 2 {
		return fmt.Errorf("changing-flag baseline must hold exactly two variants")
	}
	fp := *base
	if c.changing {
		for _, v := range fp.Variants {
			if v.Key != base.DefaultVariant {
				fp.DefaultVariant = v.Key
				break
			}
		}
	}
	payload, err := json.Marshal(fp)
	if err != nil {
		return err
	}
	return c.flipt.updateResource(ctx, resourceRequest{
		NamespaceKey: namespaceKey,
		Key:          "changing-flag",
		Payload:      payload,
	})
}

// resetFlag restores the baseline flag set, discarding any /change mutation,
// without an availability blip. /reset is what the TCK calls before almost
// every scenario.
func (c *control) resetFlag(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changing = false
	return c.seed(ctx)
}

func (c *control) handleStart(w http.ResponseWriter, r *http.Request) {
	if err := c.startBoot(r.Context(), r.URL.Query().Get("config")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "backend started serving the canonical flag set")
}

func (c *control) handleStop(w http.ResponseWriter, r *http.Request) {
	if err := c.stopBoot(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "backend unreachable; container still running")
}

func (c *control) handleRestart(w http.ResponseWriter, r *http.Request) {
	if err := c.stopBoot(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := c.startBoot(r.Context(), r.URL.Query().Get("config")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "backend restarted")
}

func (c *control) handleChange(w http.ResponseWriter, r *http.Request) {
	if err := c.changeFlag(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "flag configuration changed")
}

func (c *control) handleReset(w http.ResponseWriter, r *http.Request) {
	if err := c.resetFlag(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "flag configuration reset to baseline")
}

func (c *control) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := &control{flipt: newFliptAPI()}
	// The baseline is translated once, up front: a flag the translation
	// cannot express must crash the container here rather than seed
	// something the suite did not ask for later.
	baseline, err := loadBaseline()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load canonical flags:", err)
		os.Exit(1)
	}
	c.baseline = baseline
	// flipt must already be listening when the suite waits for the stack, so
	// spawn it before serving the control API. The initial flag set is seeded
	// by the suite's first /start.
	if err := c.startBoot(ctx, "default"); err != nil {
		fmt.Fprintln(os.Stderr, "initial /flipt boot:", err)
		os.Exit(1)
	}
	defer func() {
		if err := c.stopBoot(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "stop /flipt:", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /start", c.handleStart)
	mux.HandleFunc("POST /stop", c.handleStop)
	mux.HandleFunc("POST /restart", c.handleRestart)
	mux.HandleFunc("POST /change", c.handleChange)
	mux.HandleFunc("POST /reset", c.handleReset)
	mux.HandleFunc("GET /healthz", c.handleHealthz)

	srv := &http.Server{Addr: controlAddr, Handler: mux}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", controlAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "control server:", err)
		stop()
		os.Exit(1)
	}

	fmt.Println("flipt testbed control server listening on", controlAddr)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "control server:", err)
			stop()
		}
	}()
	<-ctx.Done()
	cctx, cstop := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	defer cstop()
	if err := srv.Shutdown(cctx); err != nil {
		fmt.Fprintln(os.Stderr, "control server:", err)
	}
}
