// Package relation derives local, backend-neutral relationships between changed files.
// Relations are observations, never commit grouping decisions.
package relation

import (
	"errors"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/syntax"
)

type Kind string
type Class string
type Outcome string

const (
	GitRename         Kind    = "git_rename"
	SourceTest        Kind    = "source_test"
	DirectImport      Kind    = "direct_import"
	ChangedIdentifier Kind    = "changed_identifier"
	PathProximity     Kind    = "path_proximity"
	ManifestLock      Kind    = "manifest_lock"
	Hard              Class   = "hard"
	Soft              Class   = "soft"
	Matched           Outcome = "matched"
	Ambiguous         Outcome = "ambiguous"
	Unresolved        Outcome = "unresolved"
	Unsupported       Outcome = "unsupported"
)

// File contains approved, already analyzed data. Content is optional. Import
// extraction parses its whole text with the existing syntax analyzer, while
// changed identifier extraction uses only the supplied changed-hunk evidence.
type File struct {
	Change   gitstate.Change
	Evidence []syntax.Evidence
	Content  []byte
}

// Evidence preserves the exact observed path or identifier without parsing a
// free-form explanation. Related is used for pairs such as old/new paths.
type Evidence struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Related string `json:"related,omitempty"`
}

// Relation uses file IDs from the original Git snapshot. A rename is a self
// relation because Git represents its old and new paths with one file ID.
type Relation struct {
	SourceID string   `json:"source_id"`
	TargetID string   `json:"target_id"`
	Kind     Kind     `json:"kind"`
	Class    Class    `json:"class"`
	Reason   string   `json:"reason"`
	Evidence Evidence `json:"evidence"`
	Score    *float64 `json:"score,omitempty"`
}

// Observation records an extraction attempt that did not produce a relation.
// It is diagnostic data and must not be treated as a graph edge.
type Observation struct {
	SourceID     string   `json:"source_id"`
	Kind         Kind     `json:"kind"`
	Outcome      Outcome  `json:"outcome"`
	Reason       string   `json:"reason"`
	Evidence     Evidence `json:"evidence"`
	CandidateIDs []string `json:"candidate_ids,omitempty"`
}

// Result separates confirmed relations from unresolved extraction attempts.
type Result struct {
	Relations    []Relation    `json:"relations"`
	Observations []Observation `json:"observations,omitempty"`
	Hints        []Hint        `json:"hints,omitempty"`
}

// Hint groups files that share weak contextual evidence without expanding it
// into pairwise relation edges.
type Hint struct {
	Kind     Kind     `json:"kind"`
	Evidence Evidence `json:"evidence"`
	FileIDs  []string `json:"file_ids"`
}

type importObservation struct {
	spec      string
	kind      string
	supported bool
	reason    string
}

var quotedImport = regexp.MustCompile("[\"'`]([^\"'`]+)[\"'`]")
var fromImport = regexp.MustCompile(`\bfrom\s+["']([^"']+)["']`)
var sideEffectImport = regexp.MustCompile(`^import\s+["']([^"']+)["']`)
var cssImport = regexp.MustCompile(`^@import\s+(?:url\()?\s*["']([^"']+)["']`)
var pythonFrom = regexp.MustCompile(`^from\s+(\.+[A-Za-z_][A-Za-z_0-9.]*)\s+import\b`)

// Extract returns stable relations and diagnostics for attempted but
// unresolved extraction. It never mutates Git, the input files, or a grouping.
func Extract(files []File) (Result, error) {
	result := Result{Relations: []Relation{}, Observations: []Observation{}, Hints: []Hint{}}
	byID := make(map[string]File, len(files))
	byPath := make(map[string]string, len(files))
	for _, file := range files {
		id := file.Change.ID
		if id == "" {
			return Result{}, errors.New("relation input has an empty file ID")
		}
		if _, exists := byID[id]; exists {
			return Result{}, errors.New("relation input has duplicate file IDs")
		}
		byID[id] = file
		if p := currentPath(file.Change); p != "" {
			if _, exists := byPath[p]; exists {
				byPath[p] = "" // An ambiguous path must never resolve to one file.
			} else {
				byPath[p] = id
			}
		}
	}
	ordered := append([]File(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Change.ID < ordered[j].Change.ID })
	for _, file := range ordered {
		change := file.Change
		p := currentPath(change)
		if change.Status == "renamed" && validPath(change.OldPath) != "" && validPath(change.NewPath) != "" && *change.OldPath != *change.NewPath {
			result.Relations = append(result.Relations, Relation{change.ID, change.ID, GitRename, Hard, "git_reported_rename", Evidence{"git_paths", *change.OldPath, *change.NewPath}, nil})
		}
		if p == "" {
			continue
		}
		for _, candidate := range sourceCandidates(p) {
			if target := byPath[candidate]; target != "" && target != change.ID {
				result.Relations = append(result.Relations, Relation{change.ID, target, SourceTest, Soft, "matching_test_path", Evidence{"test_source_paths", p, candidate}, nil})
			} else {
				result.Observations = append(result.Observations, Observation{change.ID, SourceTest, Unresolved, "matching_source_not_in_changed_file_set", Evidence{"test_source_paths", p, candidate}, nil})
			}
		}
		for _, observed := range importObservations(file) {
			evidence := Evidence{"import_path", observed.spec, observed.kind}
			if !observed.supported {
				result.Observations = append(result.Observations, Observation{change.ID, DirectImport, Unsupported, observed.reason, evidence, nil})
				continue
			}
			target, outcome, candidates, reason := resolveImport(p, observed.spec, byPath)
			if outcome == Matched && target != change.ID {
				result.Relations = append(result.Relations, Relation{change.ID, target, DirectImport, Soft, "observed_import_path", evidence, nil})
			} else if outcome != Matched {
				result.Observations = append(result.Observations, Observation{change.ID, DirectImport, outcome, reason, evidence, candidates})
			}
		}
	}
	declarations := make(map[string][]string)
	for _, file := range ordered {
		for _, evidence := range file.Evidence {
			if evidence.Name != "" && syntax.IsDeclaration(evidence.Kind) {
				declarations[evidence.Name] = appendUnique(declarations[evidence.Name], file.Change.ID)
			}
		}
	}
	for _, file := range ordered {
		for _, symbol := range referencedIdentifiers(file) {
			targets := declarations[symbol]
			if len(targets) == 1 && targets[0] != file.Change.ID {
				result.Relations = append(result.Relations, Relation{file.Change.ID, targets[0], ChangedIdentifier, Soft, "identifier_name_matches_changed_declaration", Evidence{"identifier_name", symbol, ""}, nil})
			} else if len(targets) > 1 {
				result.Observations = append(result.Observations, Observation{file.Change.ID, ChangedIdentifier, Ambiguous, "identifier_name_matches_multiple_changed_declarations", Evidence{"identifier_name", symbol, ""}, append([]string(nil), targets...)})
			}
		}
	}
	directoryMembers := make(map[string][]string)
	for _, file := range ordered {
		filePath := currentPath(file.Change)
		if filePath == "" {
			continue
		}
		dir := path.Dir(filePath)
		directoryMembers[dir] = append(directoryMembers[dir], file.Change.ID)
	}
	result.Relations = append(result.Relations, manifestLockRelations(ordered)...)
	for dir, members := range directoryMembers {
		if len(members) > 1 {
			result.Hints = append(result.Hints, Hint{Kind: PathProximity, Evidence: Evidence{"directory", dir, ""}, FileIDs: members})
		}
	}
	sort.Slice(result.Hints, func(i, j int) bool { return result.Hints[i].Evidence.Value < result.Hints[j].Evidence.Value })
	sort.Slice(result.Relations, func(i, j int) bool {
		a, b := result.Relations[i], result.Relations[j]
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		if a.Evidence.Type != b.Evidence.Type {
			return a.Evidence.Type < b.Evidence.Type
		}
		if a.Evidence.Value != b.Evidence.Value {
			return a.Evidence.Value < b.Evidence.Value
		}
		return a.Evidence.Related < b.Evidence.Related
	})
	unique := result.Relations[:0]
	for _, value := range result.Relations {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	result.Relations = unique
	sort.Slice(result.Observations, func(i, j int) bool {
		a, b := result.Observations[i], result.Observations[j]
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Outcome != b.Outcome {
			return a.Outcome < b.Outcome
		}
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		if a.Evidence.Type != b.Evidence.Type {
			return a.Evidence.Type < b.Evidence.Type
		}
		if a.Evidence.Value != b.Evidence.Value {
			return a.Evidence.Value < b.Evidence.Value
		}
		if a.Evidence.Related != b.Evidence.Related {
			return a.Evidence.Related < b.Evidence.Related
		}
		return strings.Join(a.CandidateIDs, "\x00") < strings.Join(b.CandidateIDs, "\x00")
	})
	result.Observations = uniqueObservations(result.Observations)
	return result, nil
}

func validPath(value *string) string {
	if value == nil || *value == "" || strings.HasPrefix(*value, "/") || strings.Contains(*value, "\\") {
		return ""
	}
	clean := path.Clean(*value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != *value {
		return ""
	}
	return clean
}

func currentPath(change gitstate.Change) string {
	if p := validPath(change.NewPath); p != "" {
		return p
	}
	return validPath(change.OldPath)
}

func sourceCandidates(p string) []string {
	dir, name := path.Split(p)
	inTestsDir := strings.HasSuffix(dir, "__tests__/")
	if inTestsDir {
		dir = strings.TrimSuffix(dir, "__tests__/")
	}
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for _, suffix := range []string{"_test", ".test", ".spec"} {
		if strings.HasSuffix(base, suffix) {
			return []string{dir + strings.TrimSuffix(base, suffix) + ext}
		}
	}
	if strings.HasPrefix(base, "test_") {
		return []string{dir + strings.TrimPrefix(base, "test_") + ext}
	}
	if inTestsDir {
		return []string{dir + name}
	}
	return nil
}

func importObservations(file File) []importObservation {
	if file.Change.Binary || file.Change.Opaque || file.Change.WorktreeKind != "file" || len(file.Content) == 0 {
		return nil
	}
	analysis := syntax.Analyze(syntax.Input{
		Language: file.Change.Language,
		Content:  file.Content,
		Hunks:    []syntax.Hunk{{StartLine: 1, EndLine: strings.Count(string(file.Content), "\n") + 1}},
	})
	if analysis.Mode != syntax.ModeStructural {
		return []importObservation{{kind: "source_file", spec: currentPath(file.Change), supported: false, reason: "syntax_evidence_unavailable"}}
	}
	bySpec := make(map[string]importObservation)
	for _, evidence := range analysis.Evidence {
		if (evidence.Role != "import" && evidence.Kind != "use_declaration") || evidence.EndByte <= evidence.StartByte || int(evidence.EndByte) > len(file.Content) {
			continue
		}
		span := strings.TrimSpace(string(file.Content[evidence.StartByte:evidence.EndByte]))
		var target string
		supported := true
		switch evidence.Kind {
		case "import_spec":
			if match := quotedImport.FindStringSubmatch(span); len(match) == 2 {
				target = match[1]
			}
		case "import_statement":
			for _, pattern := range []*regexp.Regexp{fromImport, sideEffectImport, cssImport} {
				if match := pattern.FindStringSubmatch(span); len(match) == 2 {
					target = match[1]
					break
				}
			}
		case "import_from_statement":
			if match := pythonFrom.FindStringSubmatch(span); len(match) == 2 {
				target = match[1]
			}
		case "use_declaration":
			supported = false
		default:
			supported = false
		}
		if target == "" && supported {
			supported = false
		}
		reason := ""
		if !supported {
			reason = "import_syntax_not_supported"
			if evidence.Kind == "use_declaration" {
				reason = "module_resolution_not_available"
			}
		}
		key := target
		if key == "" {
			key = span
		}
		previous, exists := bySpec[key]
		if !exists || (!previous.supported && supported) {
			bySpec[key] = importObservation{spec: key, kind: evidence.Kind, supported: supported, reason: reason}
		}
	}
	result := make([]importObservation, 0, len(bySpec))
	for _, observed := range bySpec {
		result = append(result, observed)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].spec != result[j].spec {
			return result[i].spec < result[j].spec
		}
		return result[i].kind < result[j].kind
	})
	return result
}

func resolveImport(importer, spec string, byPath map[string]string) (string, Outcome, []string, string) {
	var candidates []string
	switch {
	case strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../"):
		extensions := importExtensions(path.Ext(importer))
		if len(extensions) == 0 {
			return "", Unsupported, nil, "importer_extension_not_supported"
		}
		base := path.Clean(path.Join(path.Dir(importer), spec))
		if base == ".." || strings.HasPrefix(base, "../") {
			return "", Unresolved, nil, "import_target_outside_repository"
		}
		if ext := path.Ext(base); ext != "" {
			for _, allowed := range extensions {
				if ext == allowed {
					candidates = append(candidates, base)
					break
				}
			}
		} else {
			for _, ext := range extensions {
				candidates = append(candidates, base+ext, path.Join(base, "index"+ext))
			}
		}
	case strings.HasPrefix(spec, ".") && strings.HasSuffix(importer, ".py"):
		// Python relative imports: one dot is the current package.
		dots := len(spec) - len(strings.TrimLeft(spec, "."))
		dir := path.Dir(importer)
		for i := 1; i < dots; i++ {
			if dir == "." {
				return "", Unresolved, nil, "import_target_outside_repository"
			}
			dir = path.Dir(dir)
		}
		module := strings.ReplaceAll(strings.TrimLeft(spec, "."), ".", "/")
		base := path.Join(dir, module)
		candidates = []string{base + ".py", path.Join(base, "__init__.py")}
	default:
		// Absolute package imports need module resolution, which is outside this extractor.
		return "", Unsupported, nil, "module_resolution_not_available"
	}
	var found []string
	for _, candidate := range candidates {
		if id := byPath[candidate]; id != "" {
			found = appendUnique(found, id)
		}
	}
	sort.Strings(found)
	switch len(found) {
	case 0:
		return "", Unresolved, nil, "target_not_in_changed_file_set"
	case 1:
		return found[0], Matched, nil, ""
	default:
		return "", Ambiguous, found, "multiple_changed_targets_match_import"
	}
}

func importExtensions(importerExt string) []string {
	switch importerExt {
	case ".ts", ".tsx":
		return []string{".ts", ".tsx", ".js", ".jsx"}
	case ".js", ".jsx":
		return []string{".js", ".jsx"}
	case ".css":
		return []string{".css"}
	default:
		return nil
	}
}

func referencedIdentifiers(file File) []string {
	var result []string
	for _, evidence := range file.Evidence {
		if evidence.Kind != "identifier" && evidence.Kind != "type_identifier" && evidence.Kind != "property_identifier" {
			continue
		}
		if evidence.EndByte <= evidence.StartByte || int(evidence.EndByte) > len(file.Content) {
			continue
		}
		value := string(file.Content[evidence.StartByte:evidence.EndByte])
		if isIdentifier(value) {
			result = appendUnique(result, value)
		}
	}
	return result
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

var manifestLocks = map[string][]string{
	"package.json":   {"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock"},
	"go.mod":         {"go.sum"},
	"Cargo.toml":     {"Cargo.lock"},
	"pyproject.toml": {"poetry.lock", "uv.lock"},
}

func manifestLockRelations(files []File) []Relation {
	byDirectory := make(map[string]map[string][]string)
	for _, file := range files {
		filePath := currentPath(file.Change)
		if filePath == "" {
			continue
		}
		dir, name := path.Dir(filePath), path.Base(filePath)
		if byDirectory[dir] == nil {
			byDirectory[dir] = make(map[string][]string)
		}
		byDirectory[dir][name] = append(byDirectory[dir][name], file.Change.ID)
	}
	directories := make([]string, 0, len(byDirectory))
	for dir := range byDirectory {
		directories = append(directories, dir)
	}
	sort.Strings(directories)
	var relations []Relation
	for _, dir := range directories {
		filesByName := byDirectory[dir]
		for manifest, locks := range manifestLocks {
			for _, manifestID := range filesByName[manifest] {
				for _, lock := range locks {
					for _, lockID := range filesByName[lock] {
						relations = append(relations, Relation{manifestID, lockID, ManifestLock, Soft, "known_manifest_lock_pair", Evidence{"manifest_lock_pair", manifest, lock}, nil})
					}
				}
			}
		}
	}
	return relations
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueObservations(values []Observation) []Observation {
	if len(values) < 2 {
		return values
	}
	unique := values[:0]
	for _, value := range values {
		if len(unique) > 0 {
			previous := unique[len(unique)-1]
			if previous.SourceID == value.SourceID && previous.Kind == value.Kind && previous.Outcome == value.Outcome && previous.Reason == value.Reason && previous.Evidence == value.Evidence && strings.Join(previous.CandidateIDs, "\x00") == strings.Join(value.CandidateIDs, "\x00") {
				continue
			}
		}
		unique = append(unique, value)
	}
	return unique
}
