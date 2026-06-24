package codereadiness

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	of "github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/mod/semver"
)

const (
	defaultMinCodeVersionKey = "minCodeVersion"
	versionPrefix            = "v"
)

type Option func(*Hook)

type Hook struct {
	of.UnimplementedHook
	currentVersion     string
	comparator         func(current, required string) error
	validationRequired bool
	metadataMinVerKey  string
	logger             *slog.Logger
}

func WithComparator(comparator func(current, required string) error) Option {
	return func(h *Hook) {
		h.comparator = comparator
	}
}

func WithValidationRequired(required bool) Option {
	return func(h *Hook) {
		h.validationRequired = required
	}
}

func WithMetadataMinVerKey(key string) Option {
	return func(h *Hook) {
		h.metadataMinVerKey = key
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(h *Hook) {
		h.logger = logger
	}
}

func New(currentVersion string, opts ...Option) (*Hook, error) {
	h := &Hook{
		currentVersion:    currentVersion,
		comparator:        defaultComparator,
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
	return h, nil
}

func (h Hook) After(ctx context.Context, hookContext of.HookContext, flagEvaluationDetails of.InterfaceEvaluationDetails, hookHints of.HookHints) error {
	metadata := flagEvaluationDetails.FlagMetadata
	if metadata == nil {
		if h.validationRequired {
			return fmt.Errorf("flag metadata is nil for flag %q", hookContext.FlagKey())
		}
		h.logger.InfoContext(ctx, "flag metadata is nil, skipping validation", "flagKey", hookContext.FlagKey())
		return nil
	}

	minCodeVersionInterface, ok := metadata[h.metadataMinVerKey]
	if !ok {
		if h.validationRequired {
			return fmt.Errorf("key %q missing in flag's %q metadata", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.InfoContext(ctx, fmt.Sprintf("flag metadata %q key is missing, skipping validation", h.metadataMinVerKey), "flagKey", hookContext.FlagKey())
		return nil
	}
	minCodeVersion, ok := minCodeVersionInterface.(string)
	if !ok {
		if h.validationRequired {
			return fmt.Errorf("metadata %q is not a string for flag %q", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.InfoContext(ctx, fmt.Sprintf("flag metadata %q key is not a string, skipping validation", h.metadataMinVerKey), "flagKey", hookContext.FlagKey())
		return nil
	}
	if minCodeVersion == "" {
		if h.validationRequired {
			return fmt.Errorf("metadata %q value is empty for flag %q", h.metadataMinVerKey, hookContext.FlagKey())
		}
		h.logger.InfoContext(ctx, fmt.Sprintf("flag metadata %q value is empty, skipping validation", h.metadataMinVerKey), "flagKey", hookContext.FlagKey())
		return nil
	}
	if err := h.comparator(h.currentVersion, minCodeVersion); err != nil {
		return fmt.Errorf("current version: %q required minimum version: %q flag %q, check failed: %w", h.currentVersion, minCodeVersion, hookContext.FlagKey(), err)
	}
	return nil
}

func defaultComparator(current, required string) error {
	if !strings.HasPrefix(current, versionPrefix) {
		current = versionPrefix + current
	}
	if !strings.HasPrefix(required, versionPrefix) {
		required = versionPrefix + required
	}
	if !semver.IsValid(current) {
		return fmt.Errorf("invalid current semver: %q", current)
	}
	if !semver.IsValid(required) {
		return fmt.Errorf("invalid required semver: %q", required)
	}
	if semver.Compare(current, required) < 0 {
		return fmt.Errorf("current version %q is less than required version %q", current, required)
	}
	return nil
}
