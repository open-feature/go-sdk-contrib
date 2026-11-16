package process

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr/funcr"
	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"

	levels "github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
)

// TestLogrCorePassesErrorAndCaller checks that an error logged through flagd core reaches logr's
// Error with the zap error as err and the caller set to the real call site.
func TestLogrCorePassesErrorAndCaller(t *testing.T) {
	var lines []string
	l := funcr.New(func(_, args string) { lines = append(lines, args) }, funcr.Options{LogCaller: funcr.All})
	logger.NewLogger(zap.New(&logrCore{l: l}), false).Error("bad", zap.Error(errors.New("boom")))

	if len(lines) != 1 {
		t.Fatalf("expected one line, got %q", lines)
	}
	if !strings.Contains(lines[0], `"file"="zap_test.go"`) {
		t.Errorf("expected caller in zap_test.go, got %q", lines[0])
	}
	if !strings.Contains(lines[0], `"error"="boom"`) || strings.Contains(lines[0], `"error"=null`) {
		t.Errorf("expected the zap error passed as logr err, got %q", lines[0])
	}
	if !strings.Contains(lines[0], `"msg"="bad"`) {
		t.Errorf("expected the zap message passed to logr, got %q", lines[0])
	}
	// funcr writes a "level" key only for Info entries, so its absence proves the logr Error path.
	if strings.Contains(lines[0], `"level"=`) {
		t.Errorf("expected an Error entry without an info level key, got %q", lines[0])
	}
}

// logLines returns a zap logger backed by logrCore and funcr, and the lines funcr writes.
func logLines() (*zap.Logger, *[]string) {
	var lines []string
	l := funcr.New(func(_, args string) { lines = append(lines, args) }, funcr.Options{Verbosity: levels.Debug})
	return zap.New(&logrCore{l: l}), &lines
}

// TestLogrCoreNilStringer checks that a nil Stringer is logged as "<nil>" instead of panicking.
func TestLogrCoreNilStringer(t *testing.T) {
	z, lines := logLines()
	z.Info("tick", zap.Stringer("when", (*time.Time)(nil)))

	if len(*lines) != 1 || !strings.Contains((*lines)[0], `"when"="<nil>"`) {
		t.Errorf("expected \"when\"=\"<nil>\", got %q", *lines)
	}
}

// TestLogrCoreNamespace checks that fields after a zap.Namespace nest under it, whether the
// namespace comes with the entry or from an earlier With.
func TestLogrCoreNamespace(t *testing.T) {
	const want = `"top"="a" "sync"={"id"="inner"}`
	t.Run("entry", func(t *testing.T) {
		z, lines := logLines()
		z.Info("msg", zap.String("top", "a"), zap.Namespace("sync"), zap.String("id", "inner"))
		if len(*lines) != 1 || !strings.Contains((*lines)[0], want) {
			t.Errorf("expected %s, got %q", want, *lines)
		}
	})
	t.Run("with", func(t *testing.T) {
		z, lines := logLines()
		ns := z.With(zap.String("top", "a"), zap.Namespace("sync"))
		ns.Info("msg", zap.String("id", "inner"))
		if len(*lines) != 1 || !strings.Contains((*lines)[0], want) {
			t.Errorf("expected %s, got %q", want, *lines)
		}
	})
	t.Run("siblings", func(t *testing.T) {
		z, lines := logLines()
		ns := z.With(zap.Namespace("sync"))
		a := ns.With(zap.String("id", "a"))
		b := ns.With(zap.String("id", "b"))
		a.Info("msg")
		b.Info("msg")
		if len(*lines) != 2 || !strings.Contains((*lines)[0], `"sync"={"id"="a"}`) ||
			!strings.Contains((*lines)[1], `"sync"={"id"="b"}`) {
			t.Errorf("expected each sibling to keep its own id, got %q", *lines)
		}
	})
	t.Run("nested error", func(t *testing.T) {
		z, lines := logLines()
		z.Error("bad", zap.Namespace("sync"), zap.Error(errors.New("boom")))
		if len(*lines) != 1 || !strings.Contains((*lines)[0], `"sync"={"error"="boom"}`) {
			t.Errorf("expected the error nested under sync, got %q", *lines)
		}
	})
}
