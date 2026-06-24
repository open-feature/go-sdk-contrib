package codereadiness

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
)

func newTestHook(t *testing.T, currentVersion string, opts ...Option) *Hook {
	t.Helper()
	h, err := New(currentVersion, opts...)
	if err != nil {
		t.Fatalf("unexpected error creating hook: %v", err)
	}
	return h
}

func newTestHookContext(t *testing.T, flagKey string) of.HookContext {
	t.Helper()
	hc := of.NewHookContext(
    flagKey,
    of.Boolean,
    false,               
    of.NewClientMetadata(""),
    of.Metadata{},
    of.NewEvaluationContext("", nil),
	)
	return hc
}

func TestHook_New(t *testing.T) {
		tests := []struct {
		name       string
		currentVer string
		comparator func(current, required string) error
		logger     *slog.Logger
		expectedError string
	}{
		{
			name:          "valid constructor",
			currentVer:    "v1.0.0",
			comparator:    defaultComparator,
			logger:        slog.Default(),
			expectedError: "",
		},
		{
			name:          "missing comparator error",
			currentVer:    "v1.0.0",
			comparator:    nil,
			logger:        slog.Default(),
			expectedError: "codereadiness: comparator cannot be nil",
		},
		{
			name:          "missing logger error",
			currentVer:    "v1.0.0",
			comparator:    defaultComparator,
			logger:        nil,
			expectedError: "codereadiness: logger cannot be nil",
		},
		{
			name:          "empty current version",
			currentVer:    "",
			comparator:    defaultComparator,
			logger:        slog.Default(),
			expectedError: "codereadiness: currentVersion cannot be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := New(tc.currentVer, WithComparator(tc.comparator), WithLogger(tc.logger))
			if tc.expectedError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if h == nil {
					t.Fatal("expected non-nil Hook")
				}
			} else {
				if err == nil {
					t.Fatalf("expected error: %v", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("err = %q, want substring %q", err.Error(), tc.expectedError)
				}
			}
		})	
	}	
}

func TestHook_After_NilMetadata_ValidationNotRequired(t *testing.T) {
	handler := &testLogHandler{}
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: nil,
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasLogMessage(handler, "flag metadata is nil, skipping validation") {
		t.Errorf("expected log containing %q, recorded logs: %+v", "flag metadata is nil, skipping validation", handler.records)
	}
}

func TestHook_After_NilMetadata_ValidationRequired(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithValidationRequired(true))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: nil,
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "flag metadata is nil for flag \"testFlag\"") {
		t.Errorf("err = %q, want substring %q", err.Error(), "flag metadata is nil for flag \"\"")
	}
}

func TestHook_After_MissingMinCodeVersion_ValidationNotRequired(t *testing.T) {
	handler := &testLogHandler{}
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasLogMessage(handler, "flag metadata \"minCodeVersion\" key is missing, skipping validation") {
		t.Errorf("expected log containing %q, recorded logs: %+v", "flag metadata \"minCodeVersion\" key is missing, skipping validation", handler.records)
	}	
}

func TestHook_After_MissingMinCodeVersion_ValidationRequired(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithValidationRequired(true))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key \"minCodeVersion\" missing in flag's \"testFlag\" metadata") {
		t.Errorf("err = %q, want substring %q", err.Error(), "key \"minCodeVersion\" missing in flag's \"testFlag\" metadata")
	}
}

func TestHook_After_MinCodeVersion_IsNotAString_ValidationNotRequired(t *testing.T) {
	handler := &testLogHandler{}
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": 123,
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasLogMessage(handler, "flag metadata \"minCodeVersion\" key is not a string, skipping validation") {
		t.Errorf("expected log containing %q, recorded logs: %+v", "flag metadata \"minCodeVersion\" key is not a string, skipping validation", handler.records)
	}	
}

func TestHook_After_MinCodeVersion_IsNotAString_ValidationRequired(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithValidationRequired(true))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": 123,
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata \"minCodeVersion\" is not a string") {
		t.Errorf("err = %q, want substring %q", err.Error(), "metadata \"minCodeVersion\" is not a string")
	}
}

func TestHook_After_MinCodeVersion_Empty_ValidationNotRequired(t *testing.T) {
	handler := &testLogHandler{}
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasLogMessage(handler, "flag metadata \"minCodeVersion\" value is empty, skipping validation") {
		t.Errorf("expected log containing %q, recorded logs: %+v", "flag metadata \"minCodeVersion\" value is empty, skipping validation", handler.records)
	}	
}

func TestHook_After_MinCodeVersion_Empty_ValidationRequired(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithValidationRequired(true))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata \"minCodeVersion\" value is empty for flag \"testFlag\"") {
		t.Errorf("err = %q, want substring %q", err.Error(), "metadata \"minCodeVersion\" value is empty for flag \"testFlag\"")
	}
}

func TestHook_After_Validation_Passes(t *testing.T) {
	h := newTestHook(t, "v1.2.0")
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "v1.1.0",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHook_After_Validation_Fails(t *testing.T) {
	h := newTestHook(t, "v1.0.0")
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "v1.1.0",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "check failed") {
		t.Errorf("err = %q, want substring %q", err.Error(), "check failed")
	}
}

func TestHook_After_CustomMetadataKey_WrongKey(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithMetadataMinVerKey("customKey"), WithValidationRequired(true))

	detailsMissing := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "v1.0.0",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), detailsMissing, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key \"customKey\" missing in flag's \"testFlag\" metadata") {
		t.Errorf("err = %q, want substring %q", err.Error(), "key \"customKey\" missing in flag's \"testFlag\" metadata")
	}
}

func TestHook_After_CustomMetadataKey_ValidKey(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithMetadataMinVerKey("customKey"), WithValidationRequired(true))

	detailsValid := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"customKey": "v0.9.0",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), detailsValid, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
}

func TestHook_After_CustomComparator_Success(t *testing.T) {
	called := false
	customComparator := func(current, required string) error {
		called = true
		if current == "current" && required == "required" {
			return nil
		}
		return errors.New("custom error")
	}

	h := newTestHook(t, "current", WithComparator(customComparator))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "required",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("customComparator was not called")
	}
}

func TestHook_After_CustomComparator_Failure(t *testing.T) {
	customComparator := func(current, required string) error {
		return errors.New("custom error")
	}

	h := newTestHook(t, "other", WithComparator(customComparator))
	details := of.InterfaceEvaluationDetails{
		EvaluationDetails: of.EvaluationDetails{
			ResolutionDetail: of.ResolutionDetail{
				FlagMetadata: of.FlagMetadata{
					"minCodeVersion": "required",
				},
			},
		},
	}
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "custom error") {
		t.Errorf("err = %q, want substring %q", err.Error(), "custom error")
	}
}

func TestHook_After_DefaultComparator(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		required string
		wantErr  bool
	}{
		{"equal versions", "v1.2.3", "v1.2.3", false},
		{"greater version", "v1.3.0", "v1.2.3", false},
		{"less version", "v1.2.2", "v1.2.3", true},
		{"missing v prefix current", "1.2.3", "v1.2.3", false},
		{"missing v prefix required", "v1.2.3", "1.2.3", false},
		{"missing v prefix both", "1.3.0", "1.2.3", false},
		{"invalid current", "invalid", "v1.2.3", true},
		{"invalid required", "v1.2.3", "invalid", true},
		{"invalid both", "invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHook(t, tt.current)
			details := of.InterfaceEvaluationDetails{
				EvaluationDetails: of.EvaluationDetails{
					ResolutionDetail: of.ResolutionDetail{
						FlagMetadata: of.FlagMetadata{
							"minCodeVersion": tt.required,
						},
					},
				},
			}
			err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, of.HookHints{})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

type testLogHandler struct {
	records []slog.Record
}

func (h *testLogHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *testLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *testLogHandler) WithGroup(name string) slog.Handler        { return h }

func hasLogMessage(handler *testLogHandler, wantLogSubstr string) bool {
	for _, rec := range handler.records {
		if strings.Contains(rec.Message, wantLogSubstr) {
			return true
		}
	}
	return false
}
