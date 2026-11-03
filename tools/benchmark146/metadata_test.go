package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

type fixtureBackend struct {
	InputTokens    int
	Fixture        fixture
	Profiles       []string
	Deadline       time.Time
	Invalid        string
	MetadataGroups [][]string
	Requests       []phaseRequest
}

func (b *fixtureBackend) CountInputTokens(context.Context, []llm.Message, json.RawMessage, llm.Options) (int, error) {
	if b.InputTokens > 0 {
		return b.InputTokens, nil
	}
	return 1000, nil
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
	b.Requests = append(b.Requests, phaseRequest{m, string(s), o})
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
	if err := json.Unmarshal(benchmarkPayload(m), &payload); err != nil {
		return llm.Response{}, err
	}
	answers := map[string]any{}
	for _, g := range payload.Groups {
		if o.GenerationProfile == "bounded-category" {
			b.MetadataGroups = append(b.MetadataGroups, restoreFileIDs(g.Files, b.Fixture.Files))
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
	b := &fixtureBackend{Fixture: f, InputTokens: 61441}
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

// The archived metadata adapter retains semantic output across the explicit
// production 004 layout/budget migration; its historical wire is not 004.
type phaseRequest struct {
	Messages []llm.Message
	Schema   string
	Options  llm.Options
}

func TestFourFileMetadataMatchesProductionRequests(t *testing.T) {
	for _, f := range []fixture{fixtures()[0], contractFixtures()[0]} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		original := &fixtureBackend{Fixture: f}
		d := contextinput.Document{SchemaVersion: 1, Files: f.Files}
		prompt, err := planning.Renderer(planning.English)(d)
		if err != nil {
			t.Fatal(err)
		}
		production, err := (planning.ThreePhaseGenerator{Client: original}).Generate(ctx, contextinput.Prepared{Document: d, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, planning.English, planning.SensitiveValues{})
		if err != nil {
			t.Fatal(err)
		}
		candidate := &fixtureBackend{Fixture: f}
		archived, _, failure := finalMetadata(ctx, f, partition{Groups: f.Expected}, candidate, "mock")
		cancel()
		if failure != "" {
			t.Fatal(failure)
		}
		if !reflect.DeepEqual(production.Plan, archived) {
			t.Fatalf("%s metadata plan changed across layout migration", f.Name)
		}
		for _, req := range original.Requests[1:] {
			if len(req.Messages) != 3 || req.Options.InputLayout != "harness004" || req.Options.OutputTokens != 4096 {
				t.Fatal("004 production request contract missing")
			}
		}
		for _, req := range candidate.Requests {
			if len(req.Messages) != 2 || req.Options.InputLayout != "" {
				t.Fatal("archived adapter contract unexpectedly changed")
			}
		}
	}
}

func TestTextDiagnosticsSeparateSchemaIDsAndLengthWithoutSavingText(t *testing.T) {
	schema := json.RawMessage(`{"required":["G001","G002"]}`)
	content := `{"G001":{"scope":"scope","summary":"` + strings.Repeat("あ", 49) + `"},"unexpected":{"scope":"","summary":"private generated prose"}}`
	d := diagnoseText(schema, content)
	if !d.StrictSchema || d.MissingGroups != 1 || d.UnknownGroups != 1 || d.MaxSummaryCharacters != 49 {
		t.Fatalf("%+v", d)
	}
	encoded, _ := json.Marshal(d)
	if strings.Contains(string(encoded), "unexpected") || strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "あ") {
		t.Fatal("generated output leaked into diagnostics")
	}
	d = diagnoseText(schema, `{"G001":{"scope":"a","scope":"b","summary":"x"}}`)
	if d.StrictSchema {
		t.Fatal("duplicate property passed diagnostics")
	}
}

type auditConflictBackend struct {
	fixtureBackend
	GroupingCalls int
}

func (b *auditConflictBackend) ChatWithOptions(ctx context.Context, m []llm.Message, s json.RawMessage, o llm.Options) (llm.Response, error) {
	if o.GenerationProfile == "bounded-grouping" {
		b.GroupingCalls++
		if b.GroupingCalls == 4 {
			var payload struct {
				Files []contextinput.File `json:"files"`
			}
			if err := json.Unmarshal(benchmarkPayload(m), &payload); err != nil {
				return llm.Response{}, err
			}
			out := map[string]string{}
			for _, f := range payload.Files {
				out[f.ID] = "G001"
			}
			data, _ := json.Marshal(out)
			return llm.Response{Content: string(data), StopReason: "completed"}, nil
		}
	}
	return b.fixtureBackend.ChatWithOptions(ctx, m, s, o)
}
func TestAuditContradictionStopsBeforeMetadata(t *testing.T) {
	f := fixtures()[2]
	b := &auditConflictBackend{fixtureBackend: fixtureBackend{Fixture: f}}
	o := run(context.Background(), f, "audited", "mock", b, 48, time.Second, true)
	if !o.Unresolved || o.Reason != "transitive_contradiction" || o.MetadataExecuted || o.Exact != nil {
		t.Fatalf("%+v", o)
	}
	if b.GroupingCalls != 4 {
		t.Fatalf("unexpected calls: %d", b.GroupingCalls)
	}
}
