package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/planning"
)

func issue142NextFixtures() []fixture {
	return []fixture{
		{name: "lexical_crossdir_pairs", language: planning.English, files: []fileSpec{
			{path: "src/queue/dispatch.go", diff: "+func DispatchBackfillJob() bool { return true }\n"},
			{path: "docs/runtime/operations.md", diff: "+Document how operators monitor backfill jobs.\n"},
			{path: "src/accounts/update.go", diff: "+func UpdateProfile() bool { return true }\n"},
			{path: "docs/account/guide.md", diff: "+Document the profile update workflow.\n"},
		}, reference: [][]string{{"F001", "F002"}, {"F003", "F004"}}},
		{name: "lexical_paraphrase_gap", language: planning.English, files: []fileSpec{
			{path: "src/storage/evict.go", diff: "+func EvictStaleEntries() bool { return true }\n"},
			{path: "docs/guide/retention.md", diff: "+Describe expiration policy for temporary records.\n"},
			{path: "src/alerts/notify.go", diff: "+func NotifyAlert() bool { return true }\n"},
		}, reference: [][]string{{"F001", "F002"}, {"F003"}}},
		{name: "lexical_collision", language: planning.English, files: []fileSpec{
			{path: "src/audit/record.go", diff: "+func RecordExportAudit() bool { return true }\n"},
			{path: "docs/report/export.md", diff: "+Document the report export format.\n"},
			{path: "src/alerts/notify.go", diff: "+func NotifyAlert() bool { return true }\n"},
		}, reference: [][]string{{"F001"}, {"F002"}, {"F003"}}},
		{name: "lexical_bridge", language: planning.English, files: []fileSpec{
			{path: "src/auth/session.go", diff: "+func RefreshSessionToken() bool { return true }\n"},
			{path: "docs/security/auth.md", diff: "+Document session token renewal.\n"},
			{path: "src/metrics/usage.go", diff: "+func RecordTokenUsage() bool { return true }\n"},
			{path: "docs/ops/metrics.md", diff: "+Document API token usage counters.\n"},
		}, reference: [][]string{{"F001", "F002"}, {"F003", "F004"}}},
	}
}

type issue142NextInputs struct {
	prepared   contextinput.Prepared
	ids        []string
	baseline   []issue142Candidate
	lexical    []issue142Candidate
	groups     [][]string
	edges      []issue142LexicalEdge
	capReached bool
}

func issue142BuildNextInputs(ctx context.Context, item fixture) (issue142NextInputs, error) {
	prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
	if err != nil {
		return issue142NextInputs{}, err
	}
	raw, _, err := issue142Generate(prepared.Document, ids)
	if err != nil {
		return issue142NextInputs{}, err
	}
	baseline, err := issue142ExpandCandidates(prepared.Document, ids, raw)
	if err != nil {
		return issue142NextInputs{}, err
	}
	groups, edges, err := issue142LexicalPartition(prepared.Document, ids)
	if err != nil {
		return issue142NextInputs{}, err
	}
	result := issue142NextInputs{prepared: prepared, ids: ids, baseline: baseline,
		lexical: append([]issue142Candidate(nil), baseline...), groups: groups, edges: edges,
		capReached: len(baseline) >= issue142TournamentCap}
	duplicate := false
	for _, candidate := range baseline {
		if sameGroups(candidate.Groups, groups) {
			duplicate = true
			break
		}
	}
	if !duplicate && !result.capReached {
		result.lexical = append(result.lexical, issue142Candidate{
			ID: fmt.Sprintf("C%03d", len(baseline)+1), Groups: groups,
		})
	}
	return result, nil
}

type issue142NextGeneratorRow struct {
	Fixture              string                `json:"fixture"`
	BaselineCount        int                   `json:"baseline_count"`
	LexicalCount         int                   `json:"lexical_count"`
	BaselineGold         bool                  `json:"baseline_gold"`
	LexicalGold          bool                  `json:"lexical_gold"`
	LexicalPartitionGold bool                  `json:"lexical_partition_gold"`
	Added                *issue142Candidate    `json:"added,omitempty"`
	AddedNonGold         int                   `json:"added_non_gold"`
	CapReached           bool                  `json:"cap_reached"`
	Edges                []issue142LexicalEdge `json:"edges"`
	GoldInternalEdges    int                   `json:"gold_internal_edges"`
	GoldCrossEdges       int                   `json:"gold_cross_edges"`
}

func runIssue142NextLexical(ctx context.Context, options issue140Options) error {
	enc := json.NewEncoder(os.Stdout)
	matched := false
	for _, item := range issue142NextFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		input, err := issue142BuildNextInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		baselineGold, _ := issue142HasGold(input.baseline, item.reference)
		lexicalGold, _ := issue142HasGold(input.lexical, item.reference)
		row := issue142NextGeneratorRow{Fixture: item.name, BaselineCount: len(input.baseline),
			LexicalCount: len(input.lexical), BaselineGold: baselineGold, LexicalGold: lexicalGold,
			LexicalPartitionGold: sameGroups(input.groups, item.reference), CapReached: input.capReached,
			Edges: input.edges}
		if len(input.lexical) > len(input.baseline) {
			candidate := input.lexical[len(input.lexical)-1]
			row.Added = &candidate
			if !sameGroups(candidate.Groups, item.reference) {
				row.AddedNonGold = 1
			}
		}
		for _, edge := range input.edges {
			same := false
			for _, group := range item.reference {
				found := 0
				for _, id := range group {
					if id == edge.IDs[0] || id == edge.IDs[1] {
						found++
					}
				}
				if found == 2 {
					same = true
					break
				}
			}
			if same {
				row.GoldInternalEdges++
			} else {
				row.GoldCrossEdges++
			}
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 next lexical fixture %q", options.fixtureName)
	}
	return nil
}

type issue142NextRankingArm struct {
	Name       string                  `json:"name"`
	Candidates []issue142Candidate     `json:"candidates"`
	Calls      []issue142DiagnosticRow `json:"calls,omitempty"`
}

type issue142NextRankingRow struct {
	Fixture string                   `json:"fixture"`
	Arms    []issue142NextRankingArm `json:"arms"`
}

func runIssue142NextRanking(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("ranking fixed to MLX")
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		var err error
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	enc := json.NewEncoder(os.Stdout)
	matched := false
	for index, item := range issue142NextFixtures() {
		if options.fixtureName != "all" && options.fixtureName != item.name {
			continue
		}
		matched = true
		input, err := issue142BuildNextInputs(ctx, item)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		row := issue142NextRankingRow{Fixture: item.name, Arms: []issue142NextRankingArm{
			{Name: "baseline", Candidates: input.baseline},
			{Name: "lexical", Candidates: input.lexical},
		}}
		if !options.describe {
			for run, reverse := range []bool{false, true} {
				order := []int{0, 1}
				if (index+run)%2 == 1 {
					order[0], order[1] = 1, 0
				}
				for _, arm := range order {
					callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					call := issue142DiagnosticCallModeWithSchemaOrder(callCtx, backend, model, "mlx",
						"next-ranking", row.Arms[arm].Name, item, input.prepared,
						row.Arms[arm].Candidates, run+1, 2048, reverse, false, true)
					cancel()
					row.Arms[arm].Calls = append(row.Arms[arm].Calls, call)
				}
			}
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 next ranking fixture %q", options.fixtureName)
	}
	return nil
}

var issue142RelationFixtures = []string{
	"verify_misleading_relation", "verify_join_present", "holdout_spurious_test_link",
}

type issue142RelationArm struct {
	Name  string                  `json:"name"`
	Calls []issue142DiagnosticRow `json:"calls,omitempty"`
}

type issue142RelationRow struct {
	Fixture    string                `json:"fixture"`
	Candidates []issue142Candidate   `json:"candidates"`
	GoldID     string                `json:"gold_id"`
	Arms       []issue142RelationArm `json:"arms"`
}

func runIssue142RelationRemoval(ctx context.Context, options issue140Options) error {
	if options.backendName != "mlx" && !options.describe {
		return fmt.Errorf("relation removal fixed to MLX")
	}
	var backend llm.OptionsBackend
	model := ""
	if !options.describe {
		var err error
		backend, model, err = openBackend(options.backendName, options.helper, options.ollamaModel, options.modelSpec)
		if err != nil {
			return err
		}
	}
	enc := json.NewEncoder(os.Stdout)
	matched := false
	for index, name := range issue142RelationFixtures {
		if options.fixtureName != "all" && options.fixtureName != name {
			continue
		}
		matched = true
		item := fixture{}
		for _, candidate := range issue142TournamentFixtures() {
			if candidate.name == name {
				item = candidate
				break
			}
		}
		if item.name == "" {
			return fmt.Errorf("missing fixture %s", name)
		}
		prepared, ids, err := issue140Prepare(ctx, item, issue140ManyArms[5], 1024)
		if err != nil {
			return err
		}
		if prepared.Document.RelationContext == nil {
			return fmt.Errorf("%s missing relation context", name)
		}
		raw, _, err := issue142Generate(prepared.Document, ids)
		if err != nil {
			return err
		}
		expanded, err := issue142ExpandCandidates(prepared.Document, ids, raw)
		if err != nil {
			return err
		}
		if len(expanded) < 2 {
			return fmt.Errorf("%s fewer than 2 candidates", name)
		}
		candidates := expanded[:2]
		_, goldID := issue142HasGold(candidates, item.reference)
		noRelation := prepared
		noRelation.Document.RelationContext = nil
		row := issue142RelationRow{Fixture: name, Candidates: candidates, GoldID: goldID,
			Arms: []issue142RelationArm{{Name: "with_relation"}, {Name: "without_relation"}}}
		if !options.describe {
			for run, reverse := range issue142BidirectionalOrder {
				order := []int{0, 1}
				if (index+run)%2 == 1 {
					order[0], order[1] = 1, 0
				}
				for _, arm := range order {
					callPrepared := prepared
					if arm == 1 {
						callPrepared = noRelation
					}
					callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					call := issue142DiagnosticCallModeWithSchemaOrder(callCtx, backend, model, "mlx",
						"relation-removal", row.Arms[arm].Name, item, callPrepared,
						candidates, run+1, 2048, reverse, true, true)
					cancel()
					row.Arms[arm].Calls = append(row.Arms[arm].Calls, call)
				}
			}
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("unknown Issue #142 relation fixture %q", options.fixtureName)
	}
	return nil
}
