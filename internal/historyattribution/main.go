package historyattribution

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type pathsFlag []string

func (p *pathsFlag) String() string     { return strings.Join(*p, ",") }
func (p *pathsFlag) Set(s string) error { *p = append(*p, s); return nil }
func validateInput(in Input) error {
	if in.Version != 1 || in.Root == "" || in.Head == "" {
		return failure("input.schema", "version_or_snapshot_missing")
	}
	switch in.History.Status {
	case "complete", "partial", "unavailable":
	default:
		return failure("input.schema", "invalid_history_status")
	}
	if in.History.Status == "complete" && (in.History.Error != nil || in.History.Dropped > 0) {
		return failure("input.schema", "inconsistent_history_status")
	}
	paths := map[string]bool{}
	for _, f := range in.Files {
		if !safePath(f.Path) || (f.OldPath != "" && !safePath(f.OldPath)) {
			return failure("input.schema", "unsafe_path")
		}
		if paths[f.Path] {
			return failure("input.schema", "duplicate_file")
		}
		if !safeText(f.Diff) {
			return failure("input.schema", "sensitive_content")
		}
		if f.OldPath != "" {
			paths[f.OldPath] = true
		}
		paths[f.Path] = true
	}
	identities := map[string]string{}
	for _, op := range in.History.Operations {
		if !safePath(op.Path) || (!paths[op.Path] && !paths[op.OldPath]) || (op.OldPath != "" && !safePath(op.OldPath)) || op.Thread == "" || op.Turn == "" || op.Item == "" || op.Index < 0 {
			return failure("input.schema", "invalid_operation_identity")
		}
		if op.Diff != nil && !safeText(*op.Diff) {
			return failure("input.schema", "sensitive_content")
		}
		key := operationKey(op)
		b, _ := json.Marshal(op)
		if old, ok := identities[key]; ok && old != string(b) {
			return failure("input.schema", "conflicting_duplicate_operation")
		}
		identities[key] = string(b)
	}
	for _, tr := range in.History.Turns {
		if !safeText(tr.Request) || !safeText(tr.Final) {
			return failure("input.schema", "sensitive_content")
		}
	}
	return nil
}
func loadInput(name string, unlimited ...bool) (Input, error) {
	var in Input
	if !safePath(filepath.Base(name)) {
		return in, failure("input.read", "unsafe_filename")
	}
	f, err := os.Open(name)
	if err != nil {
		return in, failure("input.read", "open_failed")
	}
	defer f.Close()
	var reader io.Reader = io.LimitReader(f, maxInputBytes+1)
	if len(unlimited) > 0 && unlimited[0] {
		reader = f
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return in, failure("input.read", "read_failed")
	}
	if len(b) > maxInputBytes && !(len(unlimited) > 0 && unlimited[0]) {
		return in, failure("input.read", "size_limit")
	}
	if json.Unmarshal(b, &in) != nil {
		return in, failure("input.decode", "invalid_json")
	}
	return in, validateInput(in)
}
func outputOutside(root, name string) (string, error) {
	if name == "" {
		return "", failure("output.path", "required")
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", failure("output.path", "invalid")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", failure("output.path", "parent_unavailable")
	}
	dest := filepath.Join(parent, filepath.Base(abs))
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", failure("git.root", "unavailable_root")
	}
	rel, err := filepath.Rel(canonical, dest)
	if err != nil || rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return "", failure("output.path", "must_be_outside_target_worktree")
	}
	return dest, nil
}
func execute(args []string) error {
	if len(args) == 0 {
		return failure("arguments", "capture_or_compare_required")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("root", "", "target repository root")
	out := fs.String("out", "", "new local artifact outside target worktree")
	input := fs.String("input", "", "fixed capture JSON")
	unlimited := fs.Bool("unlimited-history", false, "research only: no history deadline, receive ceiling or artifact size ceiling")
	projection := fs.Bool("project-history", false, "research only: write local field-allowlisted history v2; upstream response is unchanged")
	fetchMode := fs.String("history-fetch", "pipeline100x4", "paged20 baseline, paged100, paged500 (300MB per response), parallel100 (2 workers), parallel100x4, parallel100x8, pipeline100x4 (overlap listing), or full5; current-worktree threads only")
	timingLog := fs.String("history-timing-log", "", "new private JSON timing log outside target worktree; no history text or IDs")
	var paths pathsFlag
	fs.Var(&paths, "path", "explicit safe repository-relative file; repeatable")
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
		return failure("arguments", "invalid_flags")
	}
	switch args[0] {
	case "capture":
		dest, err := outputOutside(*root, *out)
		if err != nil {
			return err
		}
		if _, err = os.Lstat(dest); err == nil {
			return failure("output.write", "already_exists")
		}
		var profile *HistoryTiming
		timingDest := ""
		if *timingLog != "" {
			timingDest, err = outputOutside(*root, *timingLog)
			if err != nil {
				return err
			}
			if timingDest == dest {
				return failure("output.path", "timing_equals_capture")
			}
			if _, e := os.Lstat(timingDest); e == nil {
				return failure("output.write", "already_exists")
			}
			profile = &HistoryTiming{}
		}
		in, err := captureWithTiming(*root, paths, time.Now(), *fetchMode, profile, *unlimited)
		if profile != nil {
			if e := writeNew(timingDest, profile, true); e != nil {
				return e
			}
		}
		if err != nil {
			report := map[string]any{"error": err, "git_diff_only_fallback": true}
			if e := writeNew(dest, report); e != nil {
				return e
			}
			return err
		}
		if e := validateInput(in); e != nil {
			if w := writeNew(dest, map[string]any{"error": e, "git_diff_only_fallback": true}); w != nil {
				return w
			}
			return e
		}
		var saved any = in
		if *projection {
			projected, e := projectionInput(in)
			if e != nil {
				return e
			}
			saved = projected
		}
		if err = writeNew(dest, saved, *unlimited); err != nil {
			return err
		}
		if in.History.Status != "complete" {
			return in.History.Error
		}
		return nil
	case "compare", "compare-projected":
		var in Input
		var err error
		if args[0] == "compare-projected" {
			in, err = loadProjectedInput(*input, *unlimited)
		} else {
			in, err = loadInput(*input, *unlimited)
		}
		if err != nil {
			return err
		}
		dest, err := outputOutside(in.Root, *out)
		if err != nil {
			return err
		}
		if filepath.Clean(*input) == dest {
			return failure("output.path", "input_equals_output")
		}
		return writeNew(dest, compare(in), *unlimited)
	default:
		return failure("arguments", "unknown_command")
	}
}
func Main() {
	if err := execute(os.Args[1:]); err != nil {
		b, _ := json.Marshal(map[string]any{"error": err})
		fmt.Fprintln(os.Stderr, string(b))
		os.Exit(1)
	}
}

func loadProjectedInput(name string, unlimited bool) (Input, error) {
	if !safePath(filepath.Base(name)) {
		return Input{}, failure("input.read", "unsafe_filename")
	}
	f, err := os.Open(name)
	if err != nil {
		return Input{}, failure("input.read", "open_failed")
	}
	defer f.Close()
	var reader io.Reader = io.LimitReader(f, maxInputBytes+1)
	if unlimited {
		reader = f
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return Input{}, failure("input.read", "read_failed")
	}
	if !unlimited && len(b) > maxInputBytes {
		return Input{}, failure("input.read", "size_limit")
	}
	var p ProjectionInput
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&p) != nil {
		return Input{}, failure("input.decode", "invalid_projection_json")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return Input{}, failure("input.decode", "trailing_json")
	}
	return p.input()
}
