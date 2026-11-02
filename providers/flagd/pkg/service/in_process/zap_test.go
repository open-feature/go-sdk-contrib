package process

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"
)

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
}
