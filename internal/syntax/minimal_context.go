package syntax

// Minimal before/after attribution adapted from Issue #169. Unsupported or
// ambiguous declarations are omitted; context never replaces the raw diff.
import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
	treego "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

type MinimalContext struct {
	HunkIndex            int    `json:"hunk_index"`
	EnclosingDeclaration string `json:"enclosing_declaration"`
}
type contextAudit struct {
	Reasons       map[string]int `json:"reasons"`
	BeforeParseOK bool           `json:"before_parse_ok"`
	AfterParseOK  bool           `json:"after_parse_ok"`
	Candidates    int            `json:"candidates"`
	Retained      int            `json:"retained"`
}
type contextChangeBlock struct {
	hunk     int
	old, new []int
}

var contextHunkHeader = regexp.MustCompile(`(?m)^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

// Changed runs are bounded by context lines; counts must match every hunk header.
func contextBlocks(raw string) ([]contextChangeBlock, error) {
	out := []contextChangeBlock{}
	h := -1
	old, newLine, oldN, newN := 0, 0, 0, 0
	oldWant, newWant := 0, 0
	b := contextChangeBlock{}
	flush := func() {
		if len(b.old)+len(b.new) > 0 {
			out = append(out, b)
		}
		b = contextChangeBlock{hunk: h}
	}
	validate := func() error {
		if h >= 0 && (oldN != oldWant || newN != newWant) {
			return fmt.Errorf("hunk line count mismatch")
		}
		return nil
	}
	for _, line := range strings.Split(strings.TrimSuffix(raw, "\n"), "\n") {
		if m := contextHunkHeader.FindStringSubmatch(line); m != nil {
			flush()
			if e := validate(); e != nil {
				return nil, e
			}
			h++
			old, _ = strconv.Atoi(m[1])
			newLine, _ = strconv.Atoi(m[3])
			oldWant = 1
			newWant = 1
			if m[2] != "" {
				oldWant, _ = strconv.Atoi(m[2])
			}
			if m[4] != "" {
				newWant, _ = strconv.Atoi(m[4])
			}
			oldN = 0
			newN = 0
			b = contextChangeBlock{hunk: h}
			continue
		}
		if h < 0 {
			continue
		}
		if strings.HasPrefix(line, "\\ No newline") {
			continue
		}
		if len(line) == 0 {
			return nil, fmt.Errorf("empty unprefixed hunk line")
		}
		switch line[0] {
		case '-':
			b.old = append(b.old, old)
			old++
			oldN++
		case '+':
			b.new = append(b.new, newLine)
			newLine++
			newN++
		case ' ':
			flush()
			old++
			newLine++
			oldN++
			newN++
		default:
			return nil, fmt.Errorf("invalid hunk line")
		}
	}
	flush()
	if e := validate(); e != nil {
		return nil, e
	}
	return out, nil
}
func parseContextGo(ctx context.Context, source string) (*treesitter.Tree, bool) {
	if source == "" {
		return nil, true
	}
	p := treesitter.NewParser()
	defer p.Close()
	if p.SetLanguage(treesitter.NewLanguage(treego.Language())) != nil {
		return nil, false
	}
	data := []byte(source)
	t := p.ParseWithOptions(func(offset int, _ treesitter.Point) []byte {
		if offset < len(data) {
			return data[offset:]
		}
		return nil
	}, nil, &treesitter.ParseOptions{ProgressCallback: func(treesitter.ParseState) bool { return ctx.Err() != nil }})
	if t == nil {
		return nil, false
	}
	if t.RootNode().HasError() {
		t.Close()
		return nil, false
	}
	return t, true
}

// Plain functions retain their name; methods use normalized receiver type + '.' + name.
func contextDeclarationName(n *treesitter.Node, source string) string {
	name := n.ChildByFieldName("name")
	if name == nil {
		return ""
	}
	result := name.Utf8Text([]byte(source))
	if n.Kind() == "method_declaration" {
		r := n.ChildByFieldName("receiver")
		if r == nil {
			return ""
		}
		var typ string
		var walk func(*treesitter.Node)
		walk = func(x *treesitter.Node) {
			if x == nil {
				return
			}
			if x.Kind() == "type_identifier" {
				typ = x.Utf8Text([]byte(source))
			}
			for i := uint(0); i < x.NamedChildCount(); i++ {
				walk(x.NamedChild(i))
			}
		}
		walk(r)
		if typ == "" {
			return ""
		}
		result = typ + "." + result
	}
	return result
}
func contextOwner(ctx context.Context, t *treesitter.Tree, source string, line int) (string, string) {
	if t == nil {
		return "", "missing_ast"
	}
	lines := strings.Split(source, "\n")
	if line < 1 || line > len(lines) {
		return "", "line_out_of_range"
	}
	// Ignore indentation/trailing spaces, while requiring the full changed code line to belong to one declaration.
	code := lines[line-1]
	left := len(code) - len(strings.TrimLeft(code, " \t"))
	right := len(strings.TrimRight(code, " \t\r"))
	if left >= right {
		return "", "blank_line"
	}
	found := []*treesitter.Node{}
	var visit func(*treesitter.Node)
	visit = func(n *treesitter.Node) {
		if n == nil || ctx.Err() != nil {
			return
		}
		start, end := n.StartPosition(), n.EndPosition()
		row := uint(line - 1)
		if start.Row > row || end.Row < row {
			return
		}
		if (n.Kind() == "function_declaration" || n.Kind() == "method_declaration") && (start.Row < row || int(start.Column) <= left) && (end.Row > row || int(end.Column) >= right) {
			found = append(found, n)
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(t.RootNode())
	if len(found) != 1 {
		return "", "ambiguous_or_top_level"
	}
	name := contextDeclarationName(found[0], source)
	if name == "" || len([]byte(name)) > 128 {
		return "", "empty_or_long_name"
	}
	return name, ""
}
func extractMinimalContext(ctx context.Context, before, after, raw, language string) ([]MinimalContext, contextAudit, error) {
	a := contextAudit{Reasons: map[string]int{}}
	out := []MinimalContext{}
	bs, e := contextBlocks(raw)
	if e != nil {
		return nil, a, e
	}
	if !strings.EqualFold(language, "Go") {
		a.Reasons["unsupported_language"] = len(bs)
		return out, a, nil
	}
	bt, bok := parseContextGo(ctx, before)
	at, aok := parseContextGo(ctx, after)
	if bt != nil {
		defer bt.Close()
	}
	if at != nil {
		defer at.Close()
	}
	a.BeforeParseOK = bok
	a.AfterParseOK = aok
	seen := map[MinimalContext]bool{}
	for _, b := range bs {
		if err := ctx.Err(); err != nil {
			return nil, a, err
		}
		if (len(b.old) > 0 && !bok) || (len(b.new) > 0 && !aok) {
			a.Reasons["parse_error"]++
			continue
		}
		get := func(t *treesitter.Tree, s string, ls []int) map[string]bool {
			m := map[string]bool{}
			for _, l := range ls {
				n, r := contextOwner(ctx, t, s, l)
				if r != "" {
					a.Reasons[r]++
					continue
				}
				m[n] = true
			}
			return m
		}
		oldNames := get(bt, before, b.old)
		newNames := get(at, after, b.new)
		candidates := newNames
		if len(b.new) == 0 {
			candidates = oldNames
		}
		if len(b.old) > 0 && len(b.new) > 0 {
			same := len(oldNames) == len(newNames)
			for n := range oldNames {
				same = same && newNames[n]
			}
			if !same {
				a.Reasons["before_after_mismatch"]++
				continue
			}
		}
		for n := range candidates {
			entry := MinimalContext{b.hunk, n}
			if seen[entry] {
				a.Reasons["duplicate"]++
				continue
			}
			seen[entry] = true
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].HunkIndex != out[j].HunkIndex {
			return out[i].HunkIndex < out[j].HunkIndex
		}
		return out[i].EnclosingDeclaration < out[j].EnclosingDeclaration
	})
	a.Candidates = len(out)
	if len(out) > 32 {
		a.Reasons["entry_cap"] = len(out) - 32
		out = out[:32]
	}
	a.Retained = len(out)
	return out, a, nil
}

// ExtractMinimalContext returns only hunk indices and unambiguous Go function
// or receiver-qualified method names. The 32-entry and 128-byte name caps match
// #169; malformed diffs are rejected and unsupported languages add no context.
func ExtractMinimalContext(ctx context.Context, before, after []byte, raw, language string) ([]MinimalContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, _, err := extractMinimalContext(ctx, string(before), string(after), raw, language)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return entries, err
}
