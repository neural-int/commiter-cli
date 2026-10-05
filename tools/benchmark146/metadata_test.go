package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/llm"
)

type fixtureBackend struct {
	Fixture        fixture
	Profiles       []string
	Deadline       time.Time
	Invalid        string
	MetadataGroups [][]string
}

func (b *fixtureBackend) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (b *fixtureBackend) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return llm.Response{}, errors.New("missing deadline")
	}
	if b.Deadline.IsZero() {
		b.Deadline = deadline
	}
	if deadline != b.Deadline {
		return llm.Response{}, errors.New("deadline reset")
	}
	b.Profiles = append(b.Profiles, o.GenerationProfile)
	if o.GenerationProfile == "bounded-grouping" {
		truth := map[string]string{}
		for g, group := range b.Fixture.Expected {
			for _, id := range group {
				for _, file := range b.Fixture.Files {
					if file.ID == id {
						truth["path:"+*file.NewPath] = groupID(g)
					}
				}
			}
		}
		c := &capture{Truth: truth}
		return c.ChatWithOptions(ctx, m, s, o)
	}
	if b.Invalid != "" {
		return llm.Response{Content: b.Invalid, StopReason: "completed"}, nil
	}
	var payload struct {
		Groups []finalizedGroup `json:"groups"`
	}
	if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil {
		return llm.Response{}, err
	}
	answers := map[string]any{}
	for _, g := range payload.Groups {
		if o.GenerationProfile == "bounded-category" {
			b.MetadataGroups = append(b.MetadataGroups, g.FileIDs)
			kind := "fix"
			testOnly := true
			for _, file := range g.Files {
				testOnly = testOnly && strings.HasSuffix(*file.NewPath, "_test.go")
			}
			if testOnly {
				kind = "test"
			}
			answers[g.ID] = category{kind, "none"}
		} else {
			answers[g.ID] = commitText{"settings", "update retry bounds"}
		}
	}
	data, _ := json.Marshal(answers)
	return llm.Response{Content: string(data), StopReason: "completed"}, nil
}
func TestMetadataRunsOnlyAfterGlobalGroupingAndUsesCompleteGroups(t *testing.T) {
	for _, index := range []int{3, 6} {
		f := fixtures()[index]
		b := &fixtureBackend{Fixture: f}
		o := run(context.Background(), f, "bridge", "mock", b, 48, 10*time.Second, true)
		if o.Unresolved || !o.PlanSucceeded || !o.MetadataExecuted {
			t.Fatalf("%+v", o)
		}
		oracle := run(context.Background(), f, "bridge", "oracle", nil, 48, time.Second)
		firstMetadata := -1
		for i, profile := range b.Profiles {
			if profile != "bounded-grouping" && firstMetadata < 0 {
				firstMetadata = i
			}
			if profile == "bounded-grouping" && firstMetadata >= 0 {
				t.Fatal("membership after metadata")
			}
		}
		if firstMetadata != len(oracle.Windows) {
			t.Fatalf("metadata before all global judgments: %v", b.Profiles)
		}
		if signature(b.MetadataGroups) != signature(f.Expected) {
			t.Fatal("metadata received local windows instead of global groups")
		}
	}
}
func TestUnresolvedNeverCallsMetadata(t *testing.T) {
	f := fixtures()[1]
	for _, strategy := range []string{"graph-only", "bridge"} {
		b := &fixtureBackend{Fixture: f}
		o := run(context.Background(), f, strategy, "mock", b, 1, time.Second, true)
		if !o.Unresolved || o.PlanSucceeded || o.MetadataExecuted {
			t.Fatalf("%+v", o)
		}
		for _, profile := range b.Profiles {
			if profile != "bounded-grouping" {
				t.Fatal("metadata while unresolved")
			}
		}
	}
}
func TestMetadataFailureNeverReturnsPartialPlan(t *testing.T) {
	f := fixtures()[0]
	p := partition{Groups: f.Expected}
	for _, invalid := range []string{`{"G001":{"type":"fix","type":"feat","breaking_evidence_ref":"none"}}`, `{"G001":{"type":"fix","breaking_evidence_ref":"unresolved"},"G002":{"type":"fix","breaking_evidence_ref":"none"}}`, `{"G001":{"type":"fix","breaking_evidence_ref":"none"},"unknown":{"type":"fix","breaking_evidence_ref":"none"}}`} {
		b := &fixtureBackend{Fixture: f, Invalid: invalid}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		plan, _, failure := finalMetadata(ctx, f, p, b, "mock")
		cancel()
		if failure == "" || len(plan.Commits) != 0 {
			t.Fatalf("partial plan: %+v %s", plan, failure)
		}
	}
}
func TestOverflowAndCancellationStopBeforeBackend(t *testing.T) {
	f := fixtures()[0]
	f.Files[0].RawDiff = strings.Repeat("x", 16384)
	b := &fixtureBackend{Fixture: f}
	o := run(context.Background(), f, "bridge", "mock", b, 48, time.Second, true)
	if !o.Unresolved || len(b.Profiles) > 0 || o.PlanSucceeded {
		t.Fatalf("overflow not closed: %+v", o)
	}
	f = fixtures()[0]
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b = &fixtureBackend{Fixture: f}
	o = run(ctx, f, "bridge", "mock", b, 48, time.Second, true)
	if !o.Unresolved || len(b.Profiles) > 0 || o.PlanSucceeded {
		t.Fatalf("cancel not closed: %+v", o)
	}
	p := partition{Groups: f.Expected}
	f.Files[0].RawDiff = strings.Repeat("x", 16384)
	plan, _, failure := finalMetadata(context.Background(), f, p, nil, "oracle")
	if failure != "context_overflow" || len(plan.Commits) > 0 {
		t.Fatalf("metadata overflow: %s %+v", failure, plan)
	}
}
