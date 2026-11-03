package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"reflect"
	"testing"
	"time"
)

func TestFixedFixturesBridgeAndSparseComparison(t *testing.T) {
	for _, f := range fixtures() {
		t.Run(f.Name, func(t *testing.T) {
			o := run(context.Background(), f, "bridge", "oracle", nil, 48, time.Second)
			if o.Unresolved || !o.Complete || o.Exact == nil || !*o.Exact || *o.FM != 0 || *o.FS != 0 {
				t.Fatalf("bridge: %+v", o)
			}
			for _, w := range o.Windows {
				if len(w) > 4 {
					t.Fatal("unbounded task")
				}
			}
			s := run(context.Background(), f, "graph-only", "oracle", nil, 48, time.Second)
			if f.Name == "missing-edges-single-intent-16" && !s.Unresolved {
				t.Fatal("absence of edges became separation")
			}
			if f.Name == "cross-directory-single-intent-9" && len(o.Groups[0]) != 9 {
				t.Fatal("window boundary became group boundary")
			}
		})
	}
}
func TestReconciliationRejectsConflictsAndInvalidAssignment(t *testing.T) {
	ids := []string{"a", "b", "c"}
	tests := []struct {
		name   string
		r      []windowResult
		reason string
	}{
		{"direct conflict", []windowResult{{[]string{"a", "b", "c"}, [][]string{{"a", "b"}, {"c"}}}, {[]string{"a", "b"}, [][]string{{"a"}, {"b"}}}}, "contradictory_pair"},
		{"transitive conflict", []windowResult{{[]string{"a", "b"}, [][]string{{"a", "b"}}}, {[]string{"b", "c"}, [][]string{{"b", "c"}}}, {[]string{"a", "c"}, [][]string{{"a"}, {"c"}}}}, "transitive_contradiction"},
		{"unknown", []windowResult{{[]string{"a", "b", "c"}, [][]string{{"a", "b", "x"}}}}, "unknown_local_id"},
		{"duplicate", []windowResult{{[]string{"a", "b", "c"}, [][]string{{"a", "b"}, {"b", "c"}}}}, "duplicate_local_id"},
		{"missing local", []windowResult{{[]string{"a", "b", "c"}, [][]string{{"a", "b"}}}}, "missing_local_id"},
		{"missing global", []windowResult{{[]string{"a", "b"}, [][]string{{"a", "b"}}}}, "missing_global_id"},
		{"unknown relation", []windowResult{{[]string{"a", "b"}, [][]string{{"a", "b"}}}, {[]string{"c"}, [][]string{{"c"}}}}, "unobserved_group_relation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := reconcile(ids, tt.r)
			if p.Reason != tt.reason {
				t.Fatalf("%+v", p)
			}
		})
	}
}
func TestGraphOnlyNeverMergesSoftEdgesAndDeterministicOrdering(t *testing.T) {
	f := fixtures()[3]
	w, err := initialWindows(f.Graph)
	if err != nil {
		t.Fatal(err)
	}
	g := f.Graph
	g.Nodes = append(g.Nodes[:0:0], g.Nodes...)
	g.Edges = append(g.Edges[:0:0], g.Edges...)
	for i, j := 0, len(g.Nodes)-1; i < j; i, j = i+1, j-1 {
		g.Nodes[i], g.Nodes[j] = g.Nodes[j], g.Nodes[i]
	}
	for i, j := 0, len(g.Edges)-1; i < j; i, j = i+1, j-1 {
		g.Edges[i], g.Edges[j] = g.Edges[j], g.Edges[i]
	}
	got, err := initialWindows(g)
	if err != nil || !reflect.DeepEqual(w, got) {
		t.Fatalf("%v %v", got, err)
	}
	r := []windowResult{}
	for _, window := range w {
		groups := [][]string{}
		for _, id := range window {
			groups = append(groups, []string{id})
		}
		r = append(r, windowResult{window, groups})
	}
	p := reconcile(idsFor(g), r)
	if len(p.Groups) != 9 {
		t.Fatal("soft edge merged local separate judgments")
	}
}
func TestCallBudgetAndInvalidLocalOutputStopBeforeMetadata(t *testing.T) {
	f := fixtures()[1]
	o := run(context.Background(), f, "bridge", "oracle", nil, 1, time.Second)
	if !o.Unresolved || o.Reason != "window_budget" || o.MetadataExecuted || len(o.Groups) > 0 || len(o.Calls) != 1 {
		t.Fatalf("%+v", o)
	}
}

type invalidMembershipBackend struct {
	Response string
	Profiles []string
}

func (b *invalidMembershipBackend) Chat(context.Context, []llm.Message, json.RawMessage) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected legacy call")
}
func (b *invalidMembershipBackend) ChatWithOptions(_ context.Context, _ []llm.Message, _ json.RawMessage, o llm.Options) (llm.Response, error) {
	b.Profiles = append(b.Profiles, o.GenerationProfile)
	return llm.Response{Content: b.Response, StopReason: "completed"}, nil
}
func TestProductionStageOneRejectsMalformedMembershipBeforeReconciliation(t *testing.T) {
	f := fixtures()[0]
	w := idsFor(f.Graph)
	valid := map[string]string{}
	for _, file := range f.Files {
		valid["path:"+*file.NewPath] = "G001"
	}
	missing := map[string]string{}
	for k, v := range valid {
		missing[k] = v
	}
	delete(missing, "path:"+*f.Files[0].NewPath)
	unknown := map[string]string{}
	for k, v := range valid {
		unknown[k] = v
	}
	delete(unknown, "path:"+*f.Files[0].NewPath)
	unknown["path:unknown.go"] = "G001"
	a, _ := json.Marshal(missing)
	b, _ := json.Marshal(unknown)
	duplicate := `{"path:settings/setting00.go":"G001","path:settings/setting00.go":"G002"}`
	for _, response := range []string{string(a), string(b), duplicate} {
		backend := &invalidMembershipBackend{Response: response}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		r, _, err := local(ctx, f, w, backend)
		cancel()
		if err == nil || len(r.Groups) > 0 || len(backend.Profiles) != 1 || backend.Profiles[0] != "bounded-grouping" {
			t.Fatalf("malformed membership accepted: %+v %v %v", r, err, backend.Profiles)
		}
	}
}

func TestAuditCoversEveryPairWithinWindowBudget(t *testing.T) {
	for _, f := range fixtures() {
		o := run(context.Background(), f, "audited", "oracle", nil, 48, time.Second)
		if o.Unresolved || !o.Complete || o.Exact == nil || !*o.Exact {
			t.Fatalf("%s: %+v", f.Name, o)
		}
		if len(auditWindow(idsFor(f.Graph), o.LocalResults)) != 0 {
			t.Fatal("unobserved pair remained")
		}
	}
	f := fixtures()[3]
	o := run(context.Background(), f, "audited", "oracle", nil, 3, time.Second, true)
	if !o.Unresolved || o.Reason != "window_budget" || o.MetadataExecuted {
		t.Fatalf("budget did not stop: %+v", o)
	}
}

func TestGraphCannotOmitSelectedFiles(t *testing.T) {
	f := fixtures()[0]
	f.Graph.Nodes = f.Graph.Nodes[1:]
	b := &fixtureBackend{Fixture: f}
	o := run(context.Background(), f, "bridge", "mock", b, 48, time.Second, true)
	if o.Reason != "graph_selected_id_mismatch" || !o.Unresolved || len(b.Profiles) != 0 {
		t.Fatalf("%+v", o)
	}
}
