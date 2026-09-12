package tck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// The request sequence HTTPControl issues is normative, and until now nothing
// here tested it.
//
// Three rules in control-api.yaml constrain the order and the repetition of
// control calls rather than any single call, so none of them is visible in the
// code of one method:
//
//   - /reset is the preferred scenario-isolation primitive, because it causes no
//     availability blip and therefore cannot inject a spurious lifecycle event
//     into the next scenario. /start is the fallback, not the default.
//   - "The fallback is detected once per suite and cached." A backend that
//     answers 404 or 501 must be asked once, not once per scenario: over a few
//     hundred scenarios the difference is hundreds of pointless round trips and,
//     worse, hundreds of chances for an intermittent 404 to look like a
//     supported endpoint.
//   - "/reset MUST NOT be expected to start a backend that is currently
//     stopped." So the scenario following a disconnect is prepared with /start,
//     and a suite that reset instead would silently run its next scenario
//     against a backend that is still down.
//
// Every one of the three was previously reachable only by starting Docker and
// reading a container's logs, which means in practice they were not checked at
// all. These tests need no Docker: the control API is HTTP, so a recording
// httptest.Server is a complete stand-in for a backend, and it can answer
// things a real launchpad cannot be made to answer on demand — a 501, an
// intermittent 500, a /healthz that is unready for exactly three probes.
//
// Python and JS already test these three rules. Go and Java did not.

// fakeControlAPI is a recording stand-in for a backend's control API.
//
// It records the request line of every call — method, path and query — so a
// test can assert on the whole sequence rather than on a count, which is the
// only way to say "and nothing else was called".
type fakeControlAPI struct {
	server *httptest.Server

	mu sync.Mutex
	// calls holds "POST /reset" or "POST /start?config=default", in order.
	calls []string
	// status answers a path with a code; anything unlisted answers 200.
	status map[string]int
	// healthz, when non-empty, is consumed one entry per GET /healthz and the
	// last entry repeats forever.
	healthz []int
}

func newFakeControlAPI(t *testing.T) *fakeControlAPI {
	t.Helper()

	api := &fakeControlAPI{status: map[string]int{}}
	api.server = httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(api.server.Close)

	return api
}

func (a *fakeControlAPI) handle(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if r.URL.Path == "/healthz" {
		code := http.StatusOK
		if len(a.healthz) > 0 {
			code = a.healthz[0]
			if len(a.healthz) > 1 {
				a.healthz = a.healthz[1:]
			}
		}
		w.WriteHeader(code)
		return
	}

	line := r.Method + " " + r.URL.Path
	if r.URL.RawQuery != "" {
		line += "?" + r.URL.RawQuery
	}
	a.calls = append(a.calls, line)

	code, ok := a.status[r.URL.Path]
	if !ok {
		code = http.StatusOK
	}
	w.WriteHeader(code)
}

// answer makes path respond with code from now on.
func (a *fakeControlAPI) answer(path string, code int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status[path] = code
}

// recorded returns the control calls so far, /healthz probes excluded: they are
// a readiness wait rather than a command, and a test about the command sequence
// should not have to count them.
func (a *fakeControlAPI) recorded() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}

func (a *fakeControlAPI) control(t *testing.T, opts HTTPControlOptions) *HTTPControl {
	t.Helper()

	opts.BaseURL = a.server.URL
	control, err := NewHTTPControl(opts)
	if err != nil {
		t.Fatalf("NewHTTPControl: %v", err)
	}
	return control
}

// requireSequence asserts the exact sequence of control calls, in order.
//
// Exact rather than "contains", because every one of these rules is about a
// call that must NOT have been made.
func requireSequence(t *testing.T, api *fakeControlAPI, want ...string) {
	t.Helper()

	got := api.recorded()
	if len(got) != len(want) || strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("control call sequence\n got: %s\nwant: %s",
			strings.Join(got, " | "), strings.Join(want, " | "))
	}
}

const startDefault = "POST /start?config=default"

func TestPrepareScenarioPrefersResetOverStart(t *testing.T) {
	api := newFakeControlAPI(t)
	control := api.control(t, HTTPControlOptions{})

	for scenario := range 3 {
		if err := control.PrepareScenario(context.Background()); err != nil {
			t.Fatalf("PrepareScenario %d: %v", scenario, err)
		}
	}

	// Three scenarios, three resets, and no /start at all: the blip /start
	// causes is exactly what /reset exists to avoid, so reaching for it when
	// /reset works would make the next scenario's first event unreliable.
	requireSequence(t, api, "POST /reset", "POST /reset", "POST /reset")
}

func TestPrepareScenarioFallsBackToStartWhenResetIsNotImplemented(t *testing.T) {
	// The specification names both codes, so both are tested: a backend routing
	// unknown paths to a 404 and one that knows the path and declines it with a
	// 501 must behave identically here.
	for _, code := range []int{http.StatusNotFound, http.StatusNotImplemented} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api := newFakeControlAPI(t)
			api.answer("/reset", code)
			control := api.control(t, HTTPControlOptions{})

			if err := control.PrepareScenario(context.Background()); err != nil {
				t.Fatalf("PrepareScenario: %v", err)
			}

			// The probe is not an error and the scenario is still isolated.
			requireSequence(t, api, "POST /reset", startDefault)
		})
	}
}

func TestTheResetFallbackIsProbedOncePerSuiteAndThenRemembered(t *testing.T) {
	api := newFakeControlAPI(t)
	api.answer("/reset", http.StatusNotImplemented)
	control := api.control(t, HTTPControlOptions{})

	for scenario := range 4 {
		if err := control.PrepareScenario(context.Background()); err != nil {
			t.Fatalf("PrepareScenario %d: %v", scenario, err)
		}
	}

	// One probe, four starts. The alternative — re-probing every scenario —
	// would be four wasted round trips here and several hundred in a real run,
	// and it would let one intermittent 2xx from an unimplemented endpoint
	// convince the suite that /reset works.
	requireSequence(t, api, "POST /reset", startDefault, startDefault, startDefault, startDefault)
}

func TestADisconnectForcesStartRatherThanReset(t *testing.T) {
	api := newFakeControlAPI(t)
	control := api.control(t, HTTPControlOptions{})
	ctx := context.Background()

	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("first PrepareScenario: %v", err)
	}
	if err := control.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("PrepareScenario after the disconnect: %v", err)
	}
	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("PrepareScenario after that: %v", err)
	}

	// /reset resets flag state and is specified not to start a stopped backend,
	// so the scenario after a /stop must be prepared with /start. Resetting
	// instead would leave the backend down and run the next scenario against
	// nothing — and because /reset would answer 200 regardless, the suite would
	// not notice.
	//
	// The last call matters as much as the third: the outage is over once the
	// backend has been started, so the preference for /reset comes straight
	// back rather than being lost for the rest of the suite.
	requireSequence(t, api,
		"POST /reset",
		"POST /stop",
		startDefault,
		"POST /reset",
	)
}

func TestReconnectEndsTheOutageSoTheNextScenarioResets(t *testing.T) {
	api := newFakeControlAPI(t)
	control := api.control(t, HTTPControlOptions{})
	ctx := context.Background()

	if err := control.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if err := control.Reconnect(ctx); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("PrepareScenario: %v", err)
	}

	// Reconnect is a /start, which is also a reset, so the scenario after it is
	// prepared with /reset and not with a second /start. A stale outage flag
	// here would cost every later scenario a process restart it does not need.
	requireSequence(t, api, "POST /stop", startDefault, "POST /reset")
}

func TestAResetThatFailsIsNotTreatedAsAMissingEndpoint(t *testing.T) {
	api := newFakeControlAPI(t)
	api.answer("/reset", http.StatusInternalServerError)
	control := api.control(t, HTTPControlOptions{})

	err := control.PrepareScenario(context.Background())
	if err == nil {
		t.Fatal("PrepareScenario succeeded on a 500 from /reset")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("the error does not name the status: %v", err)
	}

	// 404 and 501 are the two documented ways to say "not implemented". Any
	// other code is a backend that is broken, and falling back to /start would
	// paper over it for the rest of the suite — the scenarios would keep
	// passing while the endpoint under test stayed broken.
	requireSequence(t, api, "POST /reset")
}

func TestTheBackendConfigurationTravelsOnEveryStart(t *testing.T) {
	api := newFakeControlAPI(t)
	api.answer("/reset", http.StatusNotFound)
	control := api.control(t, HTTPControlOptions{BackendConfiguration: "custom-config"})
	ctx := context.Background()

	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("PrepareScenario: %v", err)
	}
	if err := control.Reconnect(ctx); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}

	// Both paths to /start carry it. A reconnect that started the default
	// configuration instead would restore a different flag baseline than the
	// suite began with, and the scenario after the outage would fail on a value
	// rather than on the availability it is about.
	requireSequence(t, api,
		"POST /reset",
		"POST /start?config=custom-config",
		"POST /start?config=custom-config",
	)
}

func TestChangeFlagCallsChangeAndNothingElse(t *testing.T) {
	api := newFakeControlAPI(t)
	control := api.control(t, HTTPControlOptions{})

	if err := control.ChangeFlag(context.Background()); err != nil {
		t.Fatalf("ChangeFlag: %v", err)
	}

	// In particular it does not reset first: a mutation is specified to persist
	// until the next /start or /reset, and a scenario that changes a flag then
	// asserts the new value depends on that.
	requireSequence(t, api, "POST /change")
}

func TestAwaitReadyTreatsAMissingHealthzAsReady(t *testing.T) {
	// GET /healthz is optional in the control API, so a control port answering
	// HTTP but not that path is ready enough: the TCP wait the stack already
	// passed is the documented fallback. Requiring 200 would make the suite
	// unusable against the reference launchpad, which has no /healthz.
	for _, code := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api := newFakeControlAPI(t)
			api.healthz = []int{code}
			control := api.control(t, HTTPControlOptions{})

			if err := control.AwaitReady(context.Background(), 2*time.Second); err != nil {
				t.Fatalf("AwaitReady: %v", err)
			}
			requireSequence(t, api) // a readiness probe is not a command
		})
	}
}

func TestAwaitReadyRetriesUntilTheControlAPIAnswers(t *testing.T) {
	api := newFakeControlAPI(t)
	api.healthz = []int{
		http.StatusServiceUnavailable,
		http.StatusServiceUnavailable,
		http.StatusOK,
	}
	control := api.control(t, HTTPControlOptions{})

	// The one wait in the suite that is a wait rather than an assertion. A
	// launchpad accepts connections slightly before it will act on one, and
	// this is where that window is absorbed — not in a fixed sleep after every
	// control call, which is what it replaced.
	if err := control.AwaitReady(context.Background(), 5*time.Second); err != nil {
		t.Fatalf("AwaitReady: %v", err)
	}
}

func TestAwaitReadyGivesUpWithAMessageNamingTheLastFailure(t *testing.T) {
	api := newFakeControlAPI(t)
	api.healthz = []int{http.StatusBadGateway}
	control := api.control(t, HTTPControlOptions{})

	err := control.AwaitReady(context.Background(), 300*time.Millisecond)
	if err == nil {
		t.Fatal("AwaitReady succeeded against a control API that never became ready")
	}
	for _, want := range []string{api.server.URL, "502"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the timeout message does not mention %q: %v", want, err)
		}
	}
}
