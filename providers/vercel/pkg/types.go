package vercel

// Datafile is the packed Vercel Flags datafile returned by
// https://flags.vercel.com/v1/datafile.
type Datafile struct {
	Definitions     map[string]FlagDefinition `json:"definitions"`
	Segments        map[string]Segment        `json:"segments,omitempty"`
	Environment     string                    `json:"environment"`
	ProjectID       string                    `json:"projectId,omitempty"`
	ConfigUpdatedAt any                       `json:"configUpdatedAt,omitempty"`
	Revision        int64                     `json:"revision,omitempty"`
	Digest          string                    `json:"digest,omitempty"`
}

type FlagDefinition struct {
	VariantIDs   []string       `json:"variantIds,omitempty"`
	Variants     []any          `json:"variants"`
	Environments map[string]any `json:"environments"`
	Seed         uint32         `json:"seed,omitempty"`
}

type Segment struct {
	Rules   []Rule                         `json:"rules,omitempty"`
	Include map[string]map[string][]string `json:"include,omitempty"`
	Exclude map[string]map[string][]string `json:"exclude,omitempty"`
}

type Rule struct {
	Conditions []Condition `json:"conditions"`
	Outcome    any         `json:"outcome"`
}

type Condition []any

type OutcomeType string

const (
	outcomeTypeValue   OutcomeType = "value"
	outcomeTypeSplit   OutcomeType = "split"
	outcomeTypeRollout OutcomeType = "rollout"
)

type vercelReason string

const (
	reasonPaused      vercelReason = "paused"
	reasonTargetMatch vercelReason = "target_match"
	reasonRuleMatch   vercelReason = "rule_match"
	reasonFallthrough vercelReason = "fallthrough"
	reasonError       vercelReason = "error"
)

type evaluationResult struct {
	Value        any
	Reason       vercelReason
	ErrorMessage string
	ErrorCode    string
	OutcomeType  OutcomeType
	Variant      string
}
