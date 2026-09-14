// Package codereadiness provides an OpenFeature hook that controls feature flag evaluation based on application code version and minimum required version.
package codereadiness

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/mod/semver"
)

const (
	defaultMinCodeVersionKey = "minCodeVersion"
	versionPrefix            = "v"
)

// Option is a functional option for configuring a Hook.
type Option func(*Hook)

// VersionComparator defines the interface for version comparison logic.
type VersionComparator interface {
	// Initialize is called once during Hook creation with the application's current version.
	Initialize(current string) error
	// Compare checks if the current version meets or exceeds the required version.
	Compare(required string) (bool, error)
}

// Hook validates min code version for After flag resolution.
type Hook struct {
	openfeature.UnimplementedHook
	currentVersion    string
	comparator        VersionComparator
	strictValidation  bool
	metadataMinVerKey string
	logger            *slog.Logger
}

// WithComparator sets a custom VersionComparator used to compare the current application version against the required minimum version.
func WithComparator(comparator VersionComparator) Option {
	return func(h *Hook) {
		h.comparator = comparator
	}
}

// WithStrictValidation configures whether the Hook returns an error when flag metadata or version information is missing or invalid.
func WithStrictValidation(strict bool) Option {
	return func(h *Hook) {
		h.strictValidation = strict
	}
}

// WithMetadataMinVerKey sets the metadata key used to retrieve the required minimum code version from flag metadata.
func WithMetadataMinVerKey(key string) Option {
	return func(h *Hook) {
		h.metadataMinVerKey = key
	}
}

// WithLogger sets a custom slog.Logger for logging hook events.
func WithLogger(logger *slog.Logger) Option {
	return func(h *Hook) {
		h.logger = logger
	}
}

// New creates a new CodeReadiness hook with the specified current application version and options.
func New(currentVersion string, opts ...Option) (*Hook, error) {
	h := &Hook{
		currentVersion:    currentVersion,
		comparator:        &semverComparator{},
		metadataMinVerKey: defaultMinCodeVersionKey,
		logger:            slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	if h.comparator == nil {
		return nil, errors.New("codereadiness: comparator cannot be nil")
	}
	if h.logger == nil {
		return nil, errors.New("codereadiness: logger cannot be nil")
	}
	if h.currentVersion == "" {
		return nil, errors.New("codereadiness: currentVersion cannot be empty")
	}
	if err := h.comparator.Initialize(h.currentVersion); err != nil {
		if h.strictValidation {
			return nil, fmt.Errorf("codereadiness: comparator initialization failed: %w", err)
		}
		h.logger.Debug("invalid current version values", "currentVersion", h.currentVersion, "error", err)
	}
	return h, nil
}

func (h *Hook) After(ctx context.Context, hookContext openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hookHints openfeature.HookHints) error {
	metadata := flagEvaluationDetails.FlagMetadata
	if metadata == nil {
		if h.strictValidation {
			return fmt.Errorf("flag metadata is nil for flag %q", hookContext.FlagKey())
		}
		h.logger.DebugContext(ctx, "flag metadata is nil, skipping validation", "flagKey", hookContext.FlagKey())
		return nil
	}

	minCodeVersionInterface, ok := metadata[h.metadataMinVerKey]
	if !ok {
		if h.strictValidation {
			return fmt.Errorf("key %q missing in flag's %q metadata", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.DebugContext(ctx, "flag metadata key is missing, skipping validation", "metadataKey", h.metadataMinVerKey, "flagKey", hookContext.FlagKey())
		return nil
	}
	minCodeVersion, ok := minCodeVersionInterface.(string)
	if !ok {
		if h.strictValidation {
			return fmt.Errorf("metadata %q is not a string for flag %q", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.DebugContext(ctx, "flag metadata key is not a string, skipping validation", "metadataKey", h.metadataMinVerKey, "flagKey", hookContext.FlagKey())
		return nil
	}
	if minCodeVersion == "" {
		if h.strictValidation {
			return fmt.Errorf("metadata %q value is empty for flag %q", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.DebugContext(ctx, "flag metadata key has an empty value, skipping validation", "metadataKey", h.metadataMinVerKey, "flagKey", hookContext.FlagKey())
		return nil
	}
	valid, err := h.comparator.Compare(minCodeVersion)
	if err != nil {
		if h.strictValidation {
			return fmt.Errorf("invalid version values current: %q - minimum: %q for flag %q: %w", h.currentVersion, minCodeVersion, hookContext.FlagKey(), err)
		}
		h.logger.DebugContext(ctx, "invalid version values, skipping validation", "metadataKey", h.metadataMinVerKey, "flagKey", hookContext.FlagKey(), "currentVersion", h.currentVersion, "minimumVersion", minCodeVersion, "error", err)
		return nil
	}
	if !valid {
		return fmt.Errorf("current version: %q required minimum version: %q flag %q, check failed", h.currentVersion, minCodeVersion, hookContext.FlagKey())
	}
	return nil
}

type semverComparator struct {
	current string
	currentVersionValid bool
}

func (s *semverComparator) Initialize(current string) error {
	s.current = addRequiredSemVerPrefix(current)
	if !semver.IsValid(s.current) {
		s.currentVersionValid = false
		return fmt.Errorf("invalid current semver: %q", current)
	}
	s.currentVersionValid = true
	return nil
}

func (s *semverComparator) Compare(required string) (bool, error) {
	if !s.currentVersionValid {
		return false, fmt.Errorf("invalid current semver: %q", s.current)
	}
	required = addRequiredSemVerPrefix(required)
	if !semver.IsValid(required) {
		return false, fmt.Errorf("invalid required semver: %q", required)
	}
	return semver.Compare(s.current, required) >= 0, nil
}

func addRequiredSemVerPrefix(version string) string {
	if !strings.HasPrefix(version, versionPrefix) {
		return versionPrefix + version
	}
	return version
}
