package main

import (
	"math"
	"reflect"
)

// Finite equality contrasts are facts about these sampled values. They do not
// identify a purpose, establish a commit boundary or authorize any union.
func interactionContrast(samples []snapshotValue) string {
	if len(samples) != 4 {
		return "unknown"
	}
	values := map[[2]bool]any{}
	for _, s := range samples {
		key := [2]bool{s.CalleeAfter, s.CallerAfter}
		if _, exists := values[key]; exists || !s.Known {
			return "unknown"
		}
		switch v := s.Value.(type) {
		case bool, string:
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return "unknown"
			}
		default:
			return "unknown"
		}
		values[key] = s.Value
	}
	a, b, c, d := values[[2]bool{false, false}], values[[2]bool{true, false}], values[[2]bool{false, true}], values[[2]bool{true, true}]
	eq := reflect.DeepEqual
	switch {
	case eq(a, b) && eq(a, c) && eq(a, d):
		return "no_effect"
	case eq(a, b) && eq(a, c) && !eq(a, d):
		return "joint_only_effect"
	case eq(a, b) && eq(c, d) && !eq(a, c):
		return "caller_only_effect"
	case eq(a, c) && eq(b, d) && !eq(a, b):
		return "callee_only_effect"
	default:
		return "other_combination_effect"
	}
}
