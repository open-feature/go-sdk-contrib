package process

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"
)

// TestLogrCorePassesErrorAndCaller checks that an error logged through flagd core reaches logr's
// Error with the zap error as err and the caller set to the real call site.
func TestLogrCorePassesErrorAndCaller(t *testing.T) {
	var lines []string
	l := funcr.New(func(_, args string) { lines = append(lines, args) }, funcr.Options{LogCaller: funcr.All})
	logger.NewLogger(zap.New(logrCore{l: l}), false).Error("bad", zap.Error(errors.New("boom")))

	if len(lines) != 1 {
		t.Fatalf("expected one line, got %q", lines)
	}
	if !strings.Contains(lines[0], `"file"="zap_test.go"`) {
		t.Errorf("expected caller in zap_test.go, got %q", lines[0])
	}
	if !strings.Contains(lines[0], `"error"="boom"`) || strings.Contains(lines[0], `"error"=null`) {
		t.Errorf("expected the zap error passed as logr err, got %q", lines[0])
	}
	// funcr writes a "level" key only for Info entries, so its absence proves the logr Error path.
	if strings.Contains(lines[0], `"level"=`) {
		t.Errorf("expected an Error entry without an info level key, got %q", lines[0])
	}
}
