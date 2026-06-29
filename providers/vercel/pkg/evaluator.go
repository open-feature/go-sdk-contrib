package vercel

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxRegexInputLength = 10_000

func evaluateDatafile(data *Datafile, flagKey string, defaultValue any, entities map[string]any) evaluationResult {
	if data == nil {
		return evaluationResult{
			Value:        defaultValue,
			Reason:       reasonError,
			ErrorMessage: "@vercel/flags-core: No flag definitions available. Initialize the client or provide a datafile.",
		}
	}

	definition, ok := data.Definitions[flagKey]
	if !ok {
		return evaluationResult{
			Value:        defaultValue,
			Reason:       reasonError,
			ErrorCode:    "FLAG_NOT_FOUND",
			ErrorMessage: fmt.Sprintf(`@vercel/flags-core: Definition not found for flag "%s"`, flagKey),
		}
	}

	result, err := evaluateDefinition(evaluationParams{
		defaultValue: defaultValue,
		definition:   definition,
		environment:  data.Environment,
		entities:     entities,
		segments:     data.Segments,
	}, map[string]bool{})
	if err != nil {
		return evaluationResult{
			Value:        defaultValue,
			Reason:       reasonError,
			ErrorMessage: err.Error(),
		}
	}

	return result
}

type evaluationParams struct {
	defaultValue any
	definition   FlagDefinition
	environment  string
	entities     map[string]any
	segments     map[string]Segment
}

func evaluateDefinition(params evaluationParams, visited map[string]bool) (evaluationResult, error) {
	envConfig, ok := params.definition.Environments[params.environment]
	if !ok {
		return evaluationResult{
			Value:        params.defaultValue,
			Reason:       reasonError,
			ErrorMessage: fmt.Sprintf(`Could not find envConfig for "%s"`, params.environment),
		}, nil
	}

	if idx, ok := toInt(envConfig); ok {
		value, err := getVariant(params.definition.Variants, idx)
		if err != nil {
			return evaluationResult{}, err
		}
		return evaluationResult{
			Value:       value,
			Reason:      reasonPaused,
			OutcomeType: outcomeTypeValue,
		}, nil
	}

	env, ok := asMap(envConfig)
	if !ok {
		return evaluationResult{
			Value:        params.defaultValue,
			Reason:       reasonError,
			ErrorMessage: fmt.Sprintf(`Could not find envConfig for "%s"`, params.environment),
		}, nil
	}

	if reuse, ok := env["reuse"].(string); ok {
		if visited[reuse] {
			return evaluationResult{
				Value:        params.defaultValue,
				Reason:       reasonError,
				ErrorMessage: fmt.Sprintf(`Circular environment reuse detected: "%s"`, reuse),
			}, nil
		}
		visited[params.environment] = true
		params.environment = reuse
		return evaluateDefinition(params, visited)
	}

	if targets, ok := asArray(env["targets"]); ok {
		for i, targetList := range targets {
			if matchTargetList(targetList, params) {
				value, outcomeType, err := handleOutcome(params, i)
				if err != nil {
					return evaluationResult{}, err
				}
				return evaluationResult{
					Value:       value,
					Reason:      reasonTargetMatch,
					OutcomeType: outcomeType,
				}, nil
			}
		}
	}

	for _, rule := range rulesFromAny(env["rules"]) {
		if matchConditions(rule.Conditions, params) {
			value, outcomeType, err := handleOutcome(params, rule.Outcome)
			if err != nil {
				return evaluationResult{}, err
			}
			return evaluationResult{
				Value:       value,
				Reason:      reasonRuleMatch,
				OutcomeType: outcomeType,
			}, nil
		}
	}

	fallthroughOutcome, ok := env["fallthrough"]
	if !ok {
		return evaluationResult{
			Value:        params.defaultValue,
			Reason:       reasonError,
			ErrorMessage: fmt.Sprintf(`Could not find fallthrough for "%s"`, params.environment),
		}, nil
	}

	value, outcomeType, err := handleOutcome(params, fallthroughOutcome)
	if err != nil {
		return evaluationResult{}, err
	}
	return evaluationResult{
		Value:       value,
		Reason:      reasonFallthrough,
		OutcomeType: outcomeType,
	}, nil
}

func handleOutcome(params evaluationParams, outcome any) (any, OutcomeType, error) {
	if idx, ok := toInt(outcome); ok {
		value, err := getVariant(params.definition.Variants, idx)
		return value, outcomeTypeValue, err
	}

	m, ok := asMap(outcome)
	if !ok {
		return nil, "", fmt.Errorf("@vercel/flags-core: Invalid outcome")
	}

	switch m["type"] {
	case "split":
		defaultIndex, ok := toInt(m["defaultVariant"])
		if !ok {
			return nil, "", fmt.Errorf("@vercel/flags-core: Invalid split default variant")
		}
		defaultOutcome, err := getVariant(params.definition.Variants, defaultIndex)
		if err != nil {
			return nil, "", err
		}

		lhs, ok := access(m["base"], params)
		if !ok {
			return defaultOutcome, outcomeTypeSplit, nil
		}
		lhsString, ok := lhs.(string)
		if !ok {
			return defaultOutcome, outcomeTypeSplit, nil
		}

		weights := numberSlice(m["weights"])
		if len(weights) == 0 {
			return defaultOutcome, outcomeTypeSplit, nil
		}
		var sumOfWeights float64
		for _, weight := range weights {
			sumOfWeights += weight
		}
		if sumOfWeights <= 0 {
			return defaultOutcome, outcomeTypeSplit, nil
		}

		const maxValue = 4_294_967_295
		value := float64(xxhash32String(lhsString, params.definition.Seed))
		var cumulative float64
		for i, weight := range weights {
			cumulative += (weight / sumOfWeights) * maxValue
			if value < cumulative {
				resolved, err := getVariant(params.definition.Variants, i)
				return resolved, outcomeTypeSplit, err
			}
		}
		return defaultOutcome, outcomeTypeSplit, nil

	case "rollout":
		defaultIndex, ok := toInt(m["defaultVariant"])
		if !ok {
			return nil, "", fmt.Errorf("@vercel/flags-core: Invalid rollout default variant")
		}
		defaultOutcome, err := getVariant(params.definition.Variants, defaultIndex)
		if err != nil {
			return nil, "", err
		}

		lhs, ok := access(m["base"], params)
		if !ok {
			return defaultOutcome, outcomeTypeRollout, nil
		}
		lhsString, ok := lhs.(string)
		if !ok {
			return defaultOutcome, outcomeTypeRollout, nil
		}

		startTimestamp, ok := toFloat64(m["startTimestamp"])
		if !ok {
			return nil, "", fmt.Errorf("@vercel/flags-core: Invalid rollout start timestamp")
		}
		rollFromVariant, ok := toInt(m["rollFromVariant"])
		if !ok {
			return nil, "", fmt.Errorf("@vercel/flags-core: Invalid rollout source variant")
		}
		rollToVariant, ok := toInt(m["rollToVariant"])
		if !ok {
			return nil, "", fmt.Errorf("@vercel/flags-core: Invalid rollout target variant")
		}

		elapsed := float64(time.Now().UnixMilli()) - startTimestamp
		slots := slotsFromAny(m["slots"])
		if elapsed < 0 || len(slots) == 0 {
			value, err := getVariant(params.definition.Variants, rollFromVariant)
			return value, outcomeTypeRollout, err
		}

		currentPromille := 0.0
		cumulativeDuration := 0.0
		exhausted := true
		for _, slot := range slots {
			currentPromille = slot.promille
			cumulativeDuration += slot.durationMS
			if cumulativeDuration > elapsed {
				exhausted = false
				break
			}
		}
		if exhausted {
			currentPromille = 100_000
		}
		if currentPromille <= 0 {
			value, err := getVariant(params.definition.Variants, rollFromVariant)
			return value, outcomeTypeRollout, err
		}
		if currentPromille >= 100_000 {
			value, err := getVariant(params.definition.Variants, rollToVariant)
			return value, outcomeTypeRollout, err
		}

		const maxValue = 4_294_967_295
		hash := float64(xxhash32String(lhsString, params.definition.Seed))
		threshold := (currentPromille / 100_000) * maxValue
		if hash < threshold {
			value, err := getVariant(params.definition.Variants, rollToVariant)
			return value, outcomeTypeRollout, err
		}
		value, err := getVariant(params.definition.Variants, rollFromVariant)
		return value, outcomeTypeRollout, err

	default:
		return nil, "", fmt.Errorf("@vercel/flags-core: Outcome type %v not implemented", m["type"])
	}
}

type rolloutSlot struct {
	promille   float64
	durationMS float64
}

func slotsFromAny(value any) []rolloutSlot {
	items, ok := asArray(value)
	if !ok {
		return nil
	}
	slots := make([]rolloutSlot, 0, len(items))
	for _, item := range items {
		pair, ok := asArray(item)
		if !ok || len(pair) < 2 {
			continue
		}
		promille, okPromille := toFloat64(pair[0])
		duration, okDuration := toFloat64(pair[1])
		if okPromille && okDuration {
			slots = append(slots, rolloutSlot{promille: promille, durationMS: duration})
		}
	}
	return slots
}

func getVariant(variants []any, index int) (any, error) {
	if index < 0 || index >= len(variants) {
		return nil, fmt.Errorf("@vercel/flags-core: Invalid variant index %d, variants length is %d", index, len(variants))
	}
	return normalizeJSONValue(variants[index]), nil
}

func matchTargetList(targets any, params evaluationParams) bool {
	targetMap, ok := targetListToMap(targets)
	if !ok {
		return false
	}

	for kind, attributes := range targetMap {
		for attribute, values := range attributes {
			entity, ok := access([]any{kind, attribute}, params)
			if !ok {
				continue
			}
			entityString, ok := entity.(string)
			if !ok {
				continue
			}
			for _, value := range values {
				if value == entityString {
					return true
				}
			}
		}
	}
	return false
}

func matchSegment(segment Segment, params evaluationParams) bool {
	if len(segment.Include) > 0 && matchTargetList(segment.Include, params) {
		return true
	}
	if len(segment.Exclude) > 0 && matchTargetList(segment.Exclude, params) {
		return false
	}
	if len(segment.Rules) == 0 {
		return false
	}

	for _, rule := range segment.Rules {
		if matchConditions(rule.Conditions, params) {
			return handleSegmentOutcome(rule.Outcome, params)
		}
	}
	return false
}

func handleSegmentOutcome(outcome any, params evaluationParams) bool {
	if idx, ok := toInt(outcome); ok {
		return idx == 1
	}

	m, ok := asMap(outcome)
	if !ok || m["type"] != "split" {
		return false
	}

	lhs, ok := access(m["base"], params)
	if !ok {
		return false
	}
	lhsString, ok := lhs.(string)
	if !ok {
		return false
	}

	passPromille, ok := toFloat64(m["passPromille"])
	if !ok {
		return false
	}
	if passPromille <= 0 {
		return false
	}
	if passPromille >= 100_000 {
		return true
	}

	value := xxhash32String(lhsString, params.definition.Seed) % 100_000
	return float64(value) < passPromille
}

var ignoreCaseComparators = map[string]bool{
	"eq":             true,
	"!eq":            true,
	"oneOf":          true,
	"!oneOf":         true,
	"containsAllOf":  true,
	"containsAnyOf":  true,
	"containsNoneOf": true,
	"startsWith":     true,
	"!startsWith":    true,
	"endsWith":       true,
	"!endsWith":      true,
	"contains":       true,
	"!contains":      true,
}

func matchConditions(conditions []Condition, params evaluationParams) bool {
	for _, condition := range conditions {
		if len(condition) < 2 {
			return false
		}

		lhsAccessor := condition[0]
		cmpKey, ok := condition[1].(string)
		if !ok {
			return false
		}

		var rhs any
		if len(condition) > 2 {
			rhs = condition[2]
		}

		ignoreCase := ignoreCaseComparators[cmpKey] && hasIgnoreCaseOption(condition)

		if lhsAccessor == "segment" {
			if !matchSegmentCondition(cmpKey, rhs, params) {
				return false
			}
			continue
		}

		lhs, lhsFound := access(lhsAccessor, params)
		if ignoreCase {
			lhs = lower(lhs)
			rhs = lower(rhs)
		}

		if !matchComparator(cmpKey, lhs, lhsFound, rhs) {
			return false
		}
	}
	return true
}

func hasIgnoreCaseOption(condition Condition) bool {
	if len(condition) < 4 {
		return false
	}
	switch option := condition[3].(type) {
	case string:
		return strings.Contains(option, "i")
	case map[string]any:
		if enabled, ok := option["i"].(bool); ok {
			return enabled
		}
	}
	return false
}

func matchSegmentCondition(cmp string, rhs any, params evaluationParams) bool {
	matchOne := func(segmentID string) (bool, bool) {
		segment, ok := params.segments[segmentID]
		if !ok {
			return false, false
		}
		return matchSegment(segment, params), true
	}

	switch cmp {
	case "eq":
		segmentID, ok := rhs.(string)
		if !ok {
			return false
		}
		matched, exists := matchOne(segmentID)
		return exists && matched
	case "!eq":
		segmentID, ok := rhs.(string)
		if !ok {
			return false
		}
		matched, exists := matchOne(segmentID)
		return exists && !matched
	case "oneOf":
		for _, segmentID := range stringSlice(rhs) {
			matched, exists := matchOne(segmentID)
			if exists && matched {
				return true
			}
		}
		return false
	case "!oneOf":
		segmentIDs := stringSlice(rhs)
		if len(segmentIDs) == 0 {
			return false
		}
		for _, segmentID := range segmentIDs {
			matched, exists := matchOne(segmentID)
			if !exists || matched {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func matchComparator(cmp string, lhs any, lhsFound bool, rhs any) bool {
	switch cmp {
	case "eq":
		return equal(lhs, rhs)
	case "!eq":
		return !equal(lhs, rhs)
	case "oneOf":
		return containsValue(rhs, lhs)
	case "!oneOf":
		return lhsFound && !containsValue(rhs, lhs)
	case "containsAllOf":
		return containsAll(lhs, rhs)
	case "containsAnyOf":
		return containsAny(lhs, rhs)
	case "containsNoneOf":
		return containsNone(lhs, rhs)
	case "startsWith":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && strings.HasPrefix(l, r)
	case "!startsWith":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && !strings.HasPrefix(l, r)
	case "endsWith":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && strings.HasSuffix(l, r)
	case "!endsWith":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && !strings.HasSuffix(l, r)
	case "contains":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && strings.Contains(l, r)
	case "!contains":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		return lok && rok && !strings.Contains(l, r)
	case "ex":
		return lhsFound && lhs != nil
	case "!ex":
		return !lhsFound || lhs == nil
	case "gt", "gte", "lt", "lte":
		return compare(lhs, rhs, cmp)
	case "regex", "!regex":
		matched, applicable := regexMatch(lhs, rhs)
		if !applicable {
			return false
		}
		if cmp == "!regex" {
			return !matched
		}
		return matched
	case "before", "after":
		l, lok := lhs.(string)
		r, rok := rhs.(string)
		if !lok || !rok {
			return false
		}
		left, leftOK := parseTime(l)
		right, rightOK := parseTime(r)
		if !leftOK || !rightOK {
			return false
		}
		if cmp == "before" {
			return left.Before(right)
		}
		return left.After(right)
	default:
		return false
	}
}

func access(lhs any, params evaluationParams) (any, bool) {
	path, ok := accessorPath(lhs)
	if !ok {
		return nil, false
	}

	var current any = params.entities
	for _, part := range path {
		switch typed := current.(type) {
		case map[string]any:
			key := fmt.Sprint(part)
			value, found := typed[key]
			if !found {
				return nil, false
			}
			current = value
		case []any:
			index, ok := toInt(part)
			if !ok || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return normalizeJSONValue(current), true
}

func accessorPath(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out, true
	default:
		return nil, false
	}
}

func rulesFromAny(value any) []Rule {
	switch rules := value.(type) {
	case nil:
		return nil
	case []Rule:
		return rules
	case []any:
		out := make([]Rule, 0, len(rules))
		for _, item := range rules {
			if rule, ok := ruleFromAny(item); ok {
				out = append(out, rule)
			}
		}
		return out
	default:
		return nil
	}
}

func ruleFromAny(value any) (Rule, bool) {
	switch rule := value.(type) {
	case Rule:
		return rule, true
	case map[string]any:
		return Rule{
			Conditions: conditionsFromAny(rule["conditions"]),
			Outcome:    rule["outcome"],
		}, true
	default:
		return Rule{}, false
	}
}

func conditionsFromAny(value any) []Condition {
	switch conditions := value.(type) {
	case []Condition:
		return conditions
	case []any:
		out := make([]Condition, 0, len(conditions))
		for _, item := range conditions {
			switch condition := item.(type) {
			case Condition:
				out = append(out, condition)
			case []any:
				out = append(out, Condition(condition))
			}
		}
		return out
	default:
		return nil
	}
}

func targetListToMap(value any) (map[string]map[string][]string, bool) {
	switch targets := value.(type) {
	case map[string]map[string][]string:
		return targets, true
	case map[string]any:
		out := make(map[string]map[string][]string, len(targets))
		for kind, rawAttributes := range targets {
			attrs, ok := asMap(rawAttributes)
			if !ok {
				continue
			}
			out[kind] = make(map[string][]string, len(attrs))
			for attribute, rawValues := range attrs {
				out[kind][attribute] = stringSlice(rawValues)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func asMap(value any) (map[string]any, bool) {
	m, ok := value.(map[string]any)
	return m, ok
}

func asArray(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out, true
	case []int:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out, true
	default:
		return nil, false
	}
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func numberSlice(value any) []float64 {
	items, ok := asArray(value)
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(items))
	for _, item := range items {
		if n, ok := toFloat64(item); ok {
			out = append(out, n)
		}
	}
	return out
}

func toInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if math.Trunc(typed) == typed {
			return int(typed), true
		}
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i), true
		}
		if f, err := typed.Float64(); err == nil && math.Trunc(f) == f {
			return int(f), true
		}
	case string:
		i, err := strconv.Atoi(typed)
		return i, err == nil
	}
	return 0, false
}

func toFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(typed, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i
		}
		if f, err := typed.Float64(); err == nil {
			return f
		}
		return typed.String()
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeJSONValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = normalizeJSONValue(item)
		}
		return out
	default:
		return value
	}
}

func lower(value any) any {
	switch typed := value.(type) {
	case string:
		return strings.ToLower(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = lower(item)
		}
		return out
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = strings.ToLower(item)
		}
		return out
	default:
		return value
	}
}

func equal(lhs, rhs any) bool {
	if lf, lok := toFloat64(lhs); lok {
		if rf, rok := toFloat64(rhs); rok {
			return lf == rf
		}
	}

	switch left := lhs.(type) {
	case string:
		right, ok := rhs.(string)
		return ok && left == right
	case bool:
		right, ok := rhs.(bool)
		return ok && left == right
	case nil:
		return rhs == nil
	default:
		return fmt.Sprintf("%v", lhs) == fmt.Sprintf("%v", rhs)
	}
}

func containsValue(list any, value any) bool {
	items, ok := asArray(list)
	if !ok {
		return false
	}
	for _, item := range items {
		if equal(item, value) {
			return true
		}
	}
	return false
}

func containsAll(lhs, rhs any) bool {
	right, ok := asArray(rhs)
	if !ok {
		return false
	}
	left, ok := asArray(lhs)
	if !ok {
		return false
	}
	for _, item := range right {
		if !containsValue(left, item) {
			return false
		}
	}
	return true
}

func containsAny(lhs, rhs any) bool {
	left, ok := asArray(lhs)
	if !ok {
		return false
	}
	for _, item := range left {
		if containsValue(rhs, item) {
			return true
		}
	}
	return false
}

func containsNone(lhs, rhs any) bool {
	right, ok := asArray(rhs)
	if !ok {
		return false
	}
	left, ok := asArray(lhs)
	if !ok {
		return true
	}
	for _, item := range left {
		if containsValue(right, item) {
			return false
		}
	}
	return true
}

func compare(lhs, rhs any, cmp string) bool {
	if lf, lok := toFloat64(lhs); lok {
		if rf, rok := toFloat64(rhs); rok {
			switch cmp {
			case "gt":
				return lf > rf
			case "gte":
				return lf >= rf
			case "lt":
				return lf < rf
			case "lte":
				return lf <= rf
			}
		}
	}

	ls, lok := lhs.(string)
	rs, rok := rhs.(string)
	if !lok || !rok {
		return false
	}
	switch cmp {
	case "gt":
		return ls > rs
	case "gte":
		return ls >= rs
	case "lt":
		return ls < rs
	case "lte":
		return ls <= rs
	default:
		return false
	}
}

func regexMatch(lhs, rhs any) (bool, bool) {
	input, ok := lhs.(string)
	if !ok || len(input) > maxRegexInputLength {
		return false, false
	}

	pattern, flags, ok := regexPattern(rhs)
	if !ok {
		return false, false
	}

	prefix := ""
	if strings.Contains(flags, "i") {
		prefix += "(?i)"
	}
	if strings.Contains(flags, "m") {
		prefix += "(?m)"
	}
	if strings.Contains(flags, "s") {
		prefix += "(?s)"
	}

	re, err := regexp.Compile(prefix + pattern)
	if err != nil {
		return false, false
	}
	return re.MatchString(input), true
}

func regexPattern(value any) (string, string, bool) {
	m, ok := asMap(value)
	if !ok || m["type"] != "regex" {
		return "", "", false
	}
	pattern, ok := m["pattern"].(string)
	if !ok {
		return "", "", false
	}
	flags, _ := m["flags"].(string)
	return pattern, flags, true
}

func parseTime(value string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
