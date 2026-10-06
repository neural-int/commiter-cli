package main

import "testing"

func TestObservedContrastDoesNotTurnUnknownIntoEffect(t *testing.T) {
	samples := func(values [4]any) []snapshotValue {
		out := []snapshotValue{}
		for i, bits := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			out = append(out, snapshotValue{bits[0], bits[1], true, values[i]})
		}
		return out
	}
	for _, tc := range []struct {
		name   string
		values [4]any
		want   string
	}{
		{"recorded expiry joint effect", [4]any{true, true, true, false}, "joint_only_effect"},
		{"recorded prefix caller effect", [4]any{"a", "a", "trace:a", "trace:a"}, "caller_only_effect"},
		{"callee effect", [4]any{float64(1), float64(2), float64(1), float64(2)}, "callee_only_effect"},
		{"no effect including false", [4]any{false, false, false, false}, "no_effect"},
		{"other changes", [4]any{float64(1), float64(2), float64(3), float64(4)}, "other_combination_effect"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := interactionContrast(samples(tc.values)); got != tc.want {
				t.Fatalf("got %s", got)
			}
		})
	}
	s := samples([4]any{true, true, true, false})
	s[2].Known = false
	if interactionContrast(s) != "unknown" {
		t.Fatal("unknown was treated as a value")
	}
	s = samples([4]any{true, true, true, false})
	s[2] = s[1]
	if interactionContrast(s) != "unknown" {
		t.Fatal("duplicate snapshot accepted")
	}
	if interactionContrast(s[:3]) != "unknown" {
		t.Fatal("partial observations accepted")
	}
}
