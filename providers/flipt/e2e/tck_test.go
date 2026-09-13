//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	flipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/provider"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

const (
	composeFile = "testdata/tck/docker-compose.yaml"
	fliptPort   = 8888
)

func TestFliptConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	tck.Run(t,
		tck.WithName("flipt"),
		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(fliptPort),
		tck.WithProviderFromEndpoint(func(ctx context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			baseURI := fmt.Sprintf("http://%s:%d", e.Host(), e.Port(fliptPort))
			return flipt.NewProvider(flipt.WithAddress(baseURI)), nil
		}),
		tck.WithCapabilities(
			tck.Object,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.NumericCoercion,
			tck.LargeIntegers,
		),

		tck.WithKnownDeviations(
			// string-zero-flag cannot be seeded with the empty string: Flipt
			// variant keys must be non-empty. It resolves to the "zero" key
			// instead, so the mandatory falsy-value scenario fails its
			// string-zero-flag row alone.
			tck.UntrackedDeviation("",
				"Flipt variant keys must be non-empty, so canonical string-zero-flag's empty-string value cannot be represented; it resolves to \"zero\"."),

			// Flipt's evaluation reason for a targeting rule that resolved
			// nothing is the same DEFAULT it reports for a flag with no rules
			// at all, so the provider cannot tell the two OpenFeature reasons
			// apart and reports STATIC for both. The @standard-reasons pairing
			// still passes its matching half: the TARGETING_MATCH row is what
			// Flipt reports for a rule that matched.
			tck.UntrackedDeviation(tck.StandardReasons,
				"A targeting rule that does not match reports STATIC rather than DEFAULT: Flipt's evaluation response does not distinguish a rule that matched nothing from a rule-less flag, so the two reasons cannot be told apart."),
		),
	)
}

// ofrepRunEnv gates the OFREP suite out of a default run.
const ofrepRunEnv = "OFREP_TCK_RUN"

func TestOFREPConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	if os.Getenv(ofrepRunEnv) == "" {
		t.Skipf("the OFREP conformance suite is excluded: set %s=1 to run it.", ofrepRunEnv)
	}

	tck.Run(t,
		tck.WithName("ofrep"),
		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(fliptPort),

		tck.WithProviderFromEndpoint(func(_ context.Context, endpoint tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			baseURI := fmt.Sprintf("http://%s:%d", endpoint.Host(), endpoint.Port(fliptPort))
			return ofrep.NewProvider(baseURI, ofrep.WithTimeout(5*time.Second)), nil
		}),

		tck.WithCapabilities(
			tck.Object,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.NumericCoercion,
			tck.LargeIntegers,
		),

		tck.WithReadyTimeout(30*time.Second),
		tck.WithEventTimeout(15*time.Second),

		tck.WithKnownDeviations(
			// Flipt's OFREP endpoint transparently returns its evaluation
			// reasons, and the OFREP provider does not remap them: a
			// statically-resolved flag reports flipt's DEFAULT (and
			// TARGETING_MATCH for boolean flags) instead of STATIC, and the
			// integer/float static rows additionally carry the string-value
			// TYPE_MISMATCH so their reason is ERROR.
			tck.UntrackedDeviation(tck.StandardReasons,
				"Flipt's evaluation reasons are passed through unremapped: boolean-flag reports TARGETING_MATCH and string-flag reports DEFAULT instead of STATIC for statically-resolved flags, and integer-flag and float-flag report ERROR on top of their string-value TYPE_MISMATCH."),

			// Flipt's OFREP endpoint reports variant keys as JSON strings, and
			// the OFREP provider does not coerce strings to the requested
			// numeric accessor, so integer-flag/float-flag/large-integer-flag
			// requested through their numeric accessors return TYPE_MISMATCH and
			// the code default.
			tck.UntrackedDeviation("",
				"Flipt's OFREP endpoint reports variant keys as strings and the OFREP provider does not coerce them, so numeric flags resolved as Integer or Float return TYPE_MISMATCH with the code default instead of the configured value."),

			// The falsy-value rows fail in two ways: boolean-zero-flag gets
			// flipt's MATCH reason (not STATIC) and integer-zero-flag is
			// defeated by the numeric string above; string-zero-flag carries the
			// seeded "zero" key because Flipt cannot hold an empty variant key.
			tck.UntrackedDeviation("",
				"Falsy flags: boolean-zero-flag reports TARGETING_MATCH instead of STATIC, integer-zero-flag returns TYPE_MISMATCH instead of 0, and string-zero-flag resolves to \"zero\" because Flipt variant keys must be non-empty."),

			// The lossy half of the @numeric-coercion rule passes (0.5 as an
			// integer is TYPE_MISMATCH); the lossless halves cannot, because the
			// numeric accessors reject the string 10.0/10 outright.
			tck.UntrackedDeviation(tck.NumericCoercion,
				"Lossless numeric coercion does not happen: integral-float-flag (10.0) as Integer and integer-flag (10) as Float return TYPE_MISMATCH because the OFREP provider does not coerce Flipt's string-typed values."),

			// object-flag is resolved the way any variant flag is: Flipt's
			// OFREP endpoint returns the default variant key where the suite
			// expects the structured attachment, and a String request on
			// object-flag succeeds instead of erroring.
			tck.UntrackedDeviation(tck.Object,
				"object-flag resolves to its default variant key \"template\" rather than the JSON attachment, and requesting it as a String returns \"template\" without TYPE_MISMATCH."),

			// huge-integer-flag requested as an Integer returns TYPE_MISMATCH
			// with the code default: same string-typed values as the numeric
			// deviation above, at a magnitude no coercion could rescue.
			tck.UntrackedDeviation(tck.LargeIntegers,
				"huge-integer-flag (9007199254740991) requested as Integer returns TYPE_MISMATCH because Flipt's OFREP endpoint reports the variant key as a JSON string and the OFREP provider does not coerce strings to numeric accessors."),
		),
	)
}
