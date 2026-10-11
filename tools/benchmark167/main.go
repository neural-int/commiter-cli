// benchmark167 measures authored local fixture repositories only. It cannot
// mutate a caller's repository or send model inputs outside the local helper.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
	"github.com/natsuki0413/commiter-cli/internal/planning"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type resourceState struct {
	Pressure *int     `json:"pressure_level"`
	SwapUsed *float64 `json:"swap_used_mib"`
}

var swapUsed = regexp.MustCompile(`used = ([0-9.]+)M`)

func resources() resourceState {
	r := resourceState{}
	if data, e := exec.Command("sysctl", "-n", "kern.memorystatus_vm_pressure_level").Output(); e == nil {
		if v, e := strconv.Atoi(strings.TrimSpace(string(data))); e == nil {
			r.Pressure = &v
		}
	}
	if data, e := exec.Command("sysctl", "-n", "vm.swapusage").Output(); e == nil {
		if m := swapUsed.FindStringSubmatch(string(data)); len(m) == 2 {
			if v, e := strconv.ParseFloat(m[1], 64); e == nil {
				r.SwapUsed = &v
			}
		}
	}
	return r
}
func digest(value []byte) string  { d := sha256.Sum256(value); return hex.EncodeToString(d[:]) }
func jsonDigest(value any) string { data, _ := json.Marshal(value); return digest(data) }

// Hash every fixture worktree and Git file, including index/objects/refs.
func repositoryDigest(root string) (string, error) {
	entries := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.IsDir() {
			return nil
		}
		info, e := entry.Info()
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, path)
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("unexpected fixture symlink")
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		entries[rel] = fmt.Sprintf("%o:%s", info.Mode(), digest(data))
		return nil
	})
	return jsonDigest(entries), err
}
func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_LITERAL_PATHSPECS=1", "GIT_AUTHOR_DATE=2026-10-10T00:00:00Z", "GIT_COMMITTER_DATE=2026-10-10T00:00:00Z")
	data, e := cmd.Output()
	if e != nil {
		return "", errors.New("fixture Git command failed")
	}
	return string(data), nil
}
func makeRepository(f fixture) (string, error) {
	root, e := os.MkdirTemp("", "commiter-167-fixture-")
	if e != nil {
		return "", e
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(root)
		}
	}()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "user.name", "Issue 167 fixture"}, {"config", "core.autocrlf", "false"}} {
		if _, e = git(root, args...); e != nil {
			return "", e
		}
	}
	for _, file := range f.Files {
		path := filepath.Join(root, file.Path)
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return "", e
		}
		if e = os.WriteFile(path, []byte(file.Before), 0600); e != nil {
			return "", e
		}
	}
	if _, e = git(root, "add", "--all"); e != nil {
		return "", e
	}
	if _, e = git(root, "commit", "-qm", "fixture baseline"); e != nil {
		return "", e
	}
	for _, file := range f.Files {
		if e = os.WriteFile(filepath.Join(root, file.Path), []byte(file.After), 0600); e != nil {
			return "", e
		}
	}
	failed = false
	return root, nil
}

var hunkHeader = regexp.MustCompile(`(?m)^@@ -[0-9]+(?:,[0-9]+)? \+([0-9]+)(?:,([0-9]+))? @@`)

func prepare(root string, snapshot gitstate.Snapshot) (contextinput.Prepared, int, int, error) {
	results := []syntax.ChangeResult{}
	diffBytes, diffLines := 0, 0
	for _, change := range snapshot.Changes {
		path := *change.NewPath
		raw, e := git(root, "diff", "--no-ext-diff", "--no-textconv", "--unified=0", "HEAD", "--", path)
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		diffBytes += len(raw)
		diffLines += strings.Count(raw, "\n")
		hunks := []syntax.Hunk{}
		for _, m := range hunkHeader.FindAllStringSubmatch(raw, -1) {
			start, _ := strconv.Atoi(m[1])
			count := 1
			if m[2] != "" {
				count, _ = strconv.Atoi(m[2])
			}
			hunks = append(hunks, syntax.Hunk{StartLine: start, EndLine: start + count - 1})
		}
		content, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		result, e := syntax.AnalyzeChange(syntax.ChangeInput{Change: change, Content: content, RawDiff: raw, Hunks: hunks})
		if e != nil {
			return contextinput.Prepared{}, 0, 0, e
		}
		results = append(results, result)
	}
	document, e := contextinput.Build(snapshot, results)
	if e != nil {
		return contextinput.Prepared{}, 0, 0, e
	}
	prompt, e := planning.Renderer(planning.English)(document)
	if e != nil {
		return contextinput.Prepared{}, 0, 0, e
	}
	// The actual per-profile prompts are independently bounded by both planners.
	return contextinput.Prepared{Document: document, Prompt: prompt, Budget: contextinput.Budget{ContextTokens: 16384}}, diffBytes, diffLines, nil
}

type quality struct {
	Exact bool `json:"exact"`
	FM    int  `json:"false_merge_pairs"`
	FS    int  `json:"false_split_pairs"`
}

func partitionQuality(f fixture, snapshot gitstate.Snapshot, plan planning.Plan) quality {
	purpose := map[string]int{}
	for _, file := range f.Files {
		purpose[file.Path] = file.Intent
	}
	paths := map[string]string{}
	for _, change := range snapshot.Changes {
		paths[change.ID] = *change.NewPath
	}
	groups := map[string]int{}
	for i, commit := range plan.Commits {
		for _, id := range commit.FileIDs {
			groups[id] = i
		}
	}
	q := quality{Exact: true}
	for i, a := range snapshot.Changes {
		for _, b := range snapshot.Changes[i+1:] {
			gold := purpose[paths[a.ID]] == purpose[paths[b.ID]]
			actual := groups[a.ID] == groups[b.ID]
			if gold && !actual {
				q.FS++
			}
			if !gold && actual {
				q.FM++
			}
		}
	}
	q.Exact = q.FM == 0 && q.FS == 0
	return q
}

type measurement struct {
	Workload           string        `json:"workload"`
	Files              int           `json:"files"`
	Route              string        `json:"route"`
	Repeat             int           `json:"repeat"`
	Condition          string        `json:"condition"`
	Snapshot           string        `json:"snapshot_sha256"`
	Prepared           string        `json:"prepared_sha256"`
	Wall               float64       `json:"wall_seconds"`
	Replay             float64       `json:"replay_seconds"`
	DiffBytes          int           `json:"diff_bytes"`
	DiffLines          int           `json:"diff_lines"`
	Groups             int           `json:"groups"`
	Calls              []callMetric  `json:"calls"`
	Status             string        `json:"status"`
	FailureCodes       []string      `json:"failure_codes,omitempty"`
	Diagnostic         bool          `json:"diagnostic,omitempty"`
	ProvisionalGroups  int           `json:"provisional_group_count,omitempty"`
	ProvisionalQuality *quality      `json:"provisional_quality,omitempty"`
	Quality            *quality      `json:"quality"`
	GitUnchanged       bool          `json:"git_unchanged"`
	BoundaryUnchanged  bool          `json:"boundary_unchanged"`
	Before             resourceState `json:"resource_before"`
	After              resourceState `json:"resource_after"`
}

func run(f fixture, root string, snapshot gitstate.Snapshot, p contextinput.Prepared, route string, repeat, bytes, lines int, b *measuredBackend) measurement {
	m := measurement{Workload: f.Name, Files: len(f.Files), Route: route, Repeat: repeat, Condition: "filesystem-warm", Snapshot: jsonDigest(snapshot), Prepared: jsonDigest(p.Document), DiffBytes: bytes, DiffLines: lines, Before: resources(), Status: "not_completed"}
	if repeat == 0 {
		m.Condition = "first-observed"
	}
	before, e := repositoryDigest(root)
	if e != nil {
		m.Status = "fixture_digest_failed"
		return m
	}
	started := time.Now()
	b.Calls = nil
	b.ProvisionalGroups = nil
	ctx, cancel := context.WithTimeout(context.Background(), planning.CandidateCycleTimeout)
	defer cancel()
	var preview gitstate.FileFirstPreview
	if route == "file-first" {
		replay := time.Now()
		preview, e = gitstate.PreviewFileFirst(snapshot, gitstate.Options{Context: ctx})
		m.Replay += time.Since(replay).Seconds()
	}
	var result planning.Result
	if e == nil {
		if route == "file-first" {
			result, e = (planning.FileFirstGenerator{Client: b}).Generate(ctx, p, planning.English, planning.SensitiveValues{})
		} else {
			result, e = (planning.ThreePhaseGenerator{Client: b}).Generate(ctx, p, planning.English, planning.SensitiveValues{})
		}
	}
	if e == nil && route == "file-first" {
		replay := time.Now()
		var after gitstate.FileFirstPreview
		after, e = gitstate.PreviewFileFirst(snapshot, gitstate.Options{Context: ctx})
		m.Replay += time.Since(replay).Seconds()
		if e == nil && !reflect.DeepEqual(preview, after) {
			e = errors.New("replay receipt changed")
		}
	}
	m.Wall = time.Since(started).Seconds()
	m.Calls = append([]callMetric{}, b.Calls...)
	m.After = resources()
	if b.Diagnose {
		groups := b.ProvisionalGroups
		if route == "file-first" && len(preview.Groups) == len(snapshot.Changes) {
			groups = nil
			for _, g := range preview.Groups {
				groups = append(groups, []string{g.FileID})
			}
		}
		seen := map[string]int{}
		provisional := planning.Plan{}
		for _, group := range groups {
			for _, id := range group {
				seen[id]++
			}
			provisional.Commits = append(provisional.Commits, planning.Commit{FileIDs: group})
		}
		complete := len(seen) == len(snapshot.Changes)
		for _, change := range snapshot.Changes {
			complete = complete && seen[change.ID] == 1
		}
		if complete {
			q := partitionQuality(f, snapshot, provisional)
			m.ProvisionalQuality = &q
			m.ProvisionalGroups = len(groups)
		}
	}
	after, digestErr := repositoryDigest(root)
	m.GitUnchanged = digestErr == nil && before == after
	if !m.GitUnchanged {
		m.Status = "git_changed"
		return m
	}
	if e != nil {
		m.Status = "planner_failed"
		m.FailureCodes = failureCodes(e)
		if ctx.Err() != nil {
			m.Status = "cycle_timeout"
		}
		return m
	}
	m.Groups = len(result.Plan.Commits)
	encoded, _ := json.Marshal(result.Plan)
	ids := []string{}
	for _, change := range snapshot.Changes {
		ids = append(ids, change.ID)
	}
	_, violations := planning.Validate(encoded, ids, planning.SensitiveValues{}, planning.English)
	if len(violations) != 0 {
		m.Status = "validator_failed"
		return m
	}
	if route == "file-first" {
		m.BoundaryUnchanged = len(preview.Groups) == len(result.Plan.Commits)
		for i, commit := range result.Plan.Commits {
			if len(commit.FileIDs) != 1 || i >= len(preview.Groups) || commit.FileIDs[0] != preview.Groups[i].FileID {
				m.BoundaryUnchanged = false
			}
		}
		if !m.BoundaryUnchanged {
			m.Status = "membership_changed"
			return m
		}
	}
	q := partitionQuality(f, snapshot, result.Plan)
	m.Quality = &q
	m.Status = "completed"
	return m
}
func main() {
	manifest := flag.Bool("manifest", false, "print fixed fixtures and conditions, no inference")
	helper := flag.String("helper", "", "instrumented helper executable")
	cache := flag.String("cache", "", "installed MLX model cache")
	filter := flag.String("fixture", "", "one preregistered fixture, default all")
	diagnostic := flag.Bool("diagnostic", false, "one repetition with numeric failure diagnostics for one fixed fixture")
	structures := flag.Bool("structure-only", false, "verify fixed fixture receipts and static singleton references without a model")
	regression := flag.Bool("regression-control", false, "one Three-phase cycle for a specified #143/#167 regression fixture")
	flag.Parse()
	if *diagnostic && *filter == "" {
		fmt.Fprintln(os.Stderr, "diagnostic requires one explicit preregistered fixture")
		os.Exit(2)
	}
	selected := fixtures()
	if *regression {
		if !*diagnostic || *filter == "" {
			fmt.Fprintln(os.Stderr, "regression requires diagnostic and one explicit fixture")
			os.Exit(2)
		}
		selected = regressionFixtures()
	}
	if *structures {
		if err := runStructures(selected, *filter); err != nil {
			fmt.Fprintln(os.Stderr, "structural observation failed")
			os.Exit(1)
		}
		return
	}
	if *manifest {
		data, _ := json.Marshal(selected)
		fmt.Println(string(data))
		return
	}
	if *helper == "" || *cache == "" {
		fmt.Fprintln(os.Stderr, "explicit helper and cached model required")
		os.Exit(2)
	}
	v := config.Defaults().Values
	modelPath, e := (mlxmodel.Store{Root: *cache}).Ready(mlxmodel.Spec{Repo: v.Model, Revision: v.ModelRevision, Quantization: v.ModelQuantization})
	if e != nil {
		fmt.Fprintln(os.Stderr, "pinned local model unavailable")
		os.Exit(2)
	}
	// Scope config isolation to this synthetic fixture process only.
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	b := &measuredBackend{Helper: *helper, Model: v.Model + "@" + v.ModelRevision, Path: modelPath}
	b.Diagnose = *diagnostic
	encoder := json.NewEncoder(os.Stdout)
	found := false
	for _, f := range selected {
		if *filter != "" && f.Name != *filter {
			continue
		}
		found = true
		root, e := makeRepository(f)
		if e != nil {
			fmt.Fprintln(os.Stderr, "fixture construction failed")
			os.Exit(1)
		}
		snapshot, e := gitstate.Collect(root, gitstate.Options{})
		if e != nil {
			fmt.Fprintln(os.Stderr, "fixture collection failed")
			os.Exit(1)
		}
		p, bytes, lines, e := prepare(root, snapshot)
		if e != nil {
			fmt.Fprintln(os.Stderr, "fixture preparation failed")
			os.Exit(1)
		}
		repetitions := 3
		if *diagnostic {
			repetitions = 1
		}
		for repeat := 0; repeat < repetitions; repeat++ {
			routes := []string{"file-first"}
			if len(f.Files) <= 4 {
				routes = []string{"three-phase", "file-first"}
				if repeat%2 == 1 {
					routes[0], routes[1] = routes[1], routes[0]
				}
			}
			if *regression {
				routes = []string{"three-phase"}
			}
			for _, route := range routes {
				m := run(f, root, snapshot, p, route, repeat, bytes, lines, b)
				m.Diagnostic = *diagnostic
				if *diagnostic {
					m.Condition = "diagnostic-after-main-matrix"
				}
				if *regression {
					m.Condition = "regression-control-single-cycle"
				}
				if e = encoder.Encode(m); e != nil {
					panic(e)
				}
				if !m.GitUnchanged {
					fmt.Fprintln(os.Stderr, "fixture preservation failed")
					os.Exit(1)
				}
			}
		}
		_ = os.RemoveAll(root)
	}
	if !found {
		fmt.Fprintln(os.Stderr, "unknown fixture")
		os.Exit(2)
	}
}
