package codereadiness

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

func newTestHook(t *testing.T, currentVersion string, opts ...Option) *Hook {
	t.Helper()
	h, err := New(currentVersion, opts...)
	if err != nil {
		t.Fatalf("unexpected error creating hook: %v", err)
	}
	return h
}

func newTestHookContext(t *testing.T, flagKey string) openfeature.HookContext {
	t.Helper()
	hc := openfeature.NewHookContext(
		flagKey,
		openfeature.Boolean,
		false,
		openfeature.NewClientMetadata(""),
		openfeature.Metadata{},
		openfeature.NewEvaluationContext("", nil),
	)
	return hc
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
func (h *testLogHandler) WithGroup(name string) slog.Handler       { return h }
func (h *testLogHandler) Clear()                                   { h.records = nil }

func hasLogMessage(handler *testLogHandler, wantLogSubstr string) bool {
	for _, rec := range handler.records {
		if strings.Contains(rec.Message, wantLogSubstr) {
			return true
		}
	}
	return false
}

func assertHasLogMessage(t *testing.T, handler *testLogHandler, wantLogSubstr string) {
	t.Helper()
	if !hasLogMessage(handler, wantLogSubstr) {
		t.Errorf("expected log containing %q, recorded logs: %+v", wantLogSubstr, handler.records)
	}
}

func newDetails(metadata openfeature.FlagMetadata) openfeature.InterfaceEvaluationDetails {
	return openfeature.InterfaceEvaluationDetails{
		EvaluationDetails: openfeature.EvaluationDetails{
			ResolutionDetail: openfeature.ResolutionDetail{
				FlagMetadata: metadata,
			},
		},
	}
}

func TestHook_New(t *testing.T) {
	tests := []struct {
		name          string
		currentVer    string
		comparator    VersionComparator
		logger        *slog.Logger
		strict        bool
		expectedError string
	}{
		{
			name:          "valid constructor",
			currentVer:    "v1.0.0",
			comparator:    &semverComparator{},
			logger:        slog.Default(),
			strict:        false,
			expectedError: "",
		},
		{
			name:          "missing comparator error",
			currentVer:    "v1.0.0",
			comparator:    nil,
			logger:        slog.Default(),
			strict:        false,
			expectedError: "codereadiness: comparator cannot be nil",
		},
		{
			name:          "missing logger error",
			currentVer:    "v1.0.0",
			comparator:    &semverComparator{},
			logger:        nil,
			strict:        false,
			expectedError: "codereadiness: logger cannot be nil",
		},
		{
			name:          "empty current version",
			currentVer:    "",
			comparator:    &semverComparator{},
			logger:        slog.Default(),
			strict:        false,
			expectedError: "codereadiness: currentVersion cannot be empty",
		},
		{
			name:          "invalid current version semver, strict mode",
			currentVer:    "invalid",
			comparator:    &semverComparator{},
			logger:        slog.Default(),
			strict:        true,
			expectedError: "codereadiness: comparator initialization failed",
		},
		{
			name:          "invalid current version semver, non strict mode",
			currentVer:    "invalid",
			comparator:    &semverComparator{},
			logger:        slog.Default(),
			strict:        false,
			expectedError: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := New(tc.currentVer, WithComparator(tc.comparator), WithLogger(tc.logger), WithStrictValidation(tc.strict))
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

func TestHook_After_NilMetadata_StrictValidationDisabled(t *testing.T) {
	handler := &testLogHandler{}
	t.Cleanup(handler.Clear)
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := newDetails(nil)
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertHasLogMessage(t, handler, "flag metadata is nil, skipping validation")
}

func TestHook_After_NilMetadata_StrictValidationEnabled(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithStrictValidation(true))
	details := newDetails(nil)
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "flag metadata is nil for flag \"testFlag\"") {
		t.Errorf("err = %q, want substring %q", err.Error(), "flag metadata is nil for flag \"\"")
	}
}

func TestHook_After_MissingMinCodeVersion_StrictValidationDisabled(t *testing.T) {
	handler := &testLogHandler{}
	t.Cleanup(handler.Clear)
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := newDetails(openfeature.FlagMetadata{})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertHasLogMessage(t, handler, "flag metadata key is missing, skipping validation")
}

func TestHook_After_MissingMinCodeVersion_StrictValidationEnabled(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithStrictValidation(true))
	details := newDetails(openfeature.FlagMetadata{})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key \"minCodeVersion\" missing in flag's \"testFlag\" metadata") {
		t.Errorf("err = %q, want substring %q", err.Error(), "key \"minCodeVersion\" missing in flag's \"testFlag\" metadata")
	}
}

func TestHook_After_MinCodeVersion_IsNotAString_StrictValidationDisabled(t *testing.T) {
	handler := &testLogHandler{}
	t.Cleanup(handler.Clear)
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": 123,
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertHasLogMessage(t, handler, "flag metadata key is not a string, skipping validation")
}

func TestHook_After_MinCodeVersion_IsNotAString_StrictValidationEnabled(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithStrictValidation(true))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": 123,
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata \"minCodeVersion\" is not a string") {
		t.Errorf("err = %q, want substring %q", err.Error(), "metadata \"minCodeVersion\" is not a string")
	}
}

func TestHook_After_MinCodeVersion_Empty_StrictValidationDisabled(t *testing.T) {
	handler := &testLogHandler{}
	t.Cleanup(handler.Clear)
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertHasLogMessage(t, handler, "flag metadata key has an empty value, skipping validation")
}

func TestHook_After_MinCodeVersion_Empty_StrictValidationEnabled(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithStrictValidation(true))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "metadata \"minCodeVersion\" value is empty for flag \"testFlag\"") {
		t.Errorf("err = %q, want substring %q", err.Error(), "metadata \"minCodeVersion\" value is empty for flag \"testFlag\"")
	}
}

func TestHook_After_MinCodeVersion_InvalidSemver_StrictValidationDisabled(t *testing.T) {
	handler := &testLogHandler{}
	t.Cleanup(handler.Clear)
	h := newTestHook(t, "v1.0.0", WithLogger(slog.New(handler)))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "invalid-semver",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertHasLogMessage(t, handler, "invalid version values, skipping validation")
}

func TestHook_After_MinCodeVersion_InvalidSemver_StrictValidationEnabled(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithStrictValidation(true))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "invalid-semver",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	wantSubstr := "invalid version values current: \"v1.0.0\" - minimum: \"invalid-semver\" for flag \"testFlag\""
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("err = %q, want substring %q", err.Error(), wantSubstr)
	}
}

func TestHook_After_Validation_Passes(t *testing.T) {
	h := newTestHook(t, "v1.2.0")
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "v1.1.0",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHook_After_Validation_Fails(t *testing.T) {
	h := newTestHook(t, "v1.0.0")
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "v1.1.0",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "check failed") {
		t.Errorf("err = %q, want substring %q", err.Error(), "check failed")
	}
}

func TestHook_After_CustomMetadataKey_WrongKey(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithMetadataMinVerKey("customKey"), WithStrictValidation(true))

	detailsMissing := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "v1.0.0",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), detailsMissing, openfeature.HookHints{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key \"customKey\" missing in flag's \"testFlag\" metadata") {
		t.Errorf("err = %q, want substring %q", err.Error(), "key \"customKey\" missing in flag's \"testFlag\" metadata")
	}
}

func TestHook_After_CustomMetadataKey_ValidKey(t *testing.T) {
	h := newTestHook(t, "v1.0.0", WithMetadataMinVerKey("customKey"), WithStrictValidation(true))

	detailsValid := newDetails(openfeature.FlagMetadata{
		"customKey": "v0.9.0",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), detailsValid, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

}

type mockComparator struct {
	current string
	fn      func(current, required string) (bool, error)
}

func (m *mockComparator) Initialize(current string) error {
	m.current = current
	return nil
}

func (m *mockComparator) Compare(required string) (bool, error) {
	return m.fn(m.current, required)
}

func TestHook_After_CustomComparator_Success(t *testing.T) {
	called := false
	comp := &mockComparator{
		fn: func(current, required string) (bool, error) {
			called = true
			if current == "current" && required == "required" {
				return true, nil
			}
			return false, errors.New("custom error")
		},
	}

	h := newTestHook(t, "current", WithComparator(comp))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "required",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("customComparator was not called")
	}
}

func TestHook_After_CustomComparator_Failure(t *testing.T) {
	comp := &mockComparator{
		fn: func(current, required string) (bool, error) {
			return false, errors.New("custom error")
		},
	}

	h := newTestHook(t, "other", WithComparator(comp), WithStrictValidation(true))
	details := newDetails(openfeature.FlagMetadata{
		"minCodeVersion": "required",
	})
	err := h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
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
			h, err := New(tt.current, WithStrictValidation(true))
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("unexpected error creating hook: %v", err)
			}

			details := openfeature.InterfaceEvaluationDetails{
				EvaluationDetails: openfeature.EvaluationDetails{
					ResolutionDetail: openfeature.ResolutionDetail{
						FlagMetadata: openfeature.FlagMetadata{
							"minCodeVersion": tt.required,
						},
					},
				},
			}
			err = h.After(t.Context(), newTestHookContext(t, "testFlag"), details, openfeature.HookHints{})
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
