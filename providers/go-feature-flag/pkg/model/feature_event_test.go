package model_test

import (
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	of "github.com/open-feature/go-sdk/openfeature"

	"github.com/stretchr/testify/assert"
)

func TestNewFeatureEvent(t *testing.T) {
	type args struct {
		user      of.EvaluationContext
		flagKey   string
		value     any
		variation string
		failed    bool
		version   string
		source    string
	}
	tests := []struct {
		name string
		args args
		want model.FeatureEvent
	}{
		{
			name: "anonymous user",
			args: args{
				user:      of.NewEvaluationContext("ABCD", map[string]any{"anonymous": true}),
				flagKey:   "random-key",
				value:     "YO",
				variation: "Default",
				failed:    false,
				version:   "",
				source:    "SERVER",
			},
			want: model.FeatureEvent{
				Kind: "feature", ContextKind: "anonymousUser", UserKey: "ABCD", CreationDate: time.Now().Unix(), Key: "random-key",
				Variation: "Default", Value: "YO", Default: false, Source: "SERVER",
			},
		},
		{
			name: "regular user",
			args: args{
				user:      of.NewEvaluationContext("USER-1", map[string]any{}),
				flagKey:   "my-flag",
				value:     "hello",
				variation: "True",
				failed:    false,
				version:   "",
				source:    "SERVER",
			},
			want: model.FeatureEvent{
				Kind: "feature", ContextKind: "user", UserKey: "USER-1", CreationDate: time.Now().Unix(), Key: "my-flag",
				Variation: "True", Value: "hello", Default: false, Source: "SERVER",
			},
		},
		{
			name: "failed evaluation",
			args: args{
				user:      of.NewEvaluationContext("USER-2", map[string]any{}),
				flagKey:   "my-flag",
				value:     false,
				variation: "SdkDefault",
				failed:    true,
				version:   "",
				source:    "SERVER",
			},
			want: model.FeatureEvent{
				Kind: "feature", ContextKind: "user", UserKey: "USER-2", CreationDate: time.Now().Unix(), Key: "my-flag",
				Variation: "SdkDefault", Value: false, Default: true, Source: "SERVER",
			},
		},
		{
			name: "with version",
			args: args{
				user:      of.NewEvaluationContext("USER-3", map[string]any{}),
				flagKey:   "versioned-flag",
				value:     42,
				variation: "Default",
				failed:    false,
				version:   "v1.2.3",
				source:    "PROVIDER_CACHE",
			},
			want: model.FeatureEvent{
				Kind: "feature", ContextKind: "user", UserKey: "USER-3", CreationDate: time.Now().Unix(), Key: "versioned-flag",
				Variation: "Default", Value: 42, Default: false, Version: "v1.2.3", Source: "PROVIDER_CACHE",
			},
		},
		{
			name: "non-string value types",
			args: args{
				user:      of.NewEvaluationContext("USER-4", map[string]any{}),
				flagKey:   "object-flag",
				value:     map[string]any{"key": "val"},
				variation: "Default",
				failed:    false,
				version:   "",
				source:    "SERVER",
			},
			want: model.FeatureEvent{
				Kind: "feature", ContextKind: "user", UserKey: "USER-4", CreationDate: time.Now().Unix(), Key: "object-flag",
				Variation: "Default", Value: map[string]any{"key": "val"}, Default: false, Source: "SERVER",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, model.NewFeatureEvent(tt.args.user, tt.args.flagKey, tt.args.value, tt.args.variation, tt.args.failed, tt.args.version, tt.args.source), "NewFeatureEvent(%v, %v, %v, %v, %v, %v, %V)", tt.args.user, tt.args.flagKey, tt.args.value, tt.args.variation, tt.args.failed, tt.args.version, tt.args.source)
		})
	}
}
