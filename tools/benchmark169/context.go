package main

// Research-only Go before/after Tree-sitter attribution. No product mode changes.
import (
	"fmt"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	treego "github.com/tree-sitter/tree-sitter-go/bindings/go"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type syntaxContext struct {
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
type changeBlock struct {
	hunk     int
	old, new []int
}

var fullHeader = regexp.MustCompile(`(?m)^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)

// Changed runs are bounded by context lines; counts must match every hunk header.
func blocks(raw string) ([]changeBlock, error) {
	out := []changeBlock{}
	h := -1
	old, newLine, oldN, newN := 0, 0, 0, 0
	oldWant, newWant := 0, 0
	b := changeBlock{}
	flush := func() {
		if len(b.old)+len(b.new) > 0 {
			out = append(out, b)
		}
		b = changeBlock{hunk: h}
	}
	validate := func() error {
		if h >= 0 && (oldN != oldWant || newN != newWant) {
			return fmt.Errorf("hunk line count mismatch")
		}
		return nil
	}
	for _, line := range strings.Split(strings.TrimSuffix(raw, "\n"), "\n") {
		if m := fullHeader.FindStringSubmatch(line); m != nil {
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
			b = changeBlock{hunk: h}
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
func parseGo(source string) (*treesitter.Tree, bool) {
	if source == "" {
		return nil, true
	}
	p := treesitter.NewParser()
	defer p.Close()
	if p.SetLanguage(treesitter.NewLanguage(treego.Language())) != nil {
		return nil, false
	}
	t := p.Parse([]byte(source), nil)
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
func declarationName169(n *treesitter.Node, source string) string {
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
func owner(t *treesitter.Tree, source string, line int) (string, string) {
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
		if n == nil {
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
	name := declarationName169(found[0], source)
	if name == "" || len([]byte(name)) > 128 {
		return "", "empty_or_long_name"
	}
	return name, ""
}
func extractContext(before, after, raw, language string) ([]syntaxContext, contextAudit, error) {
	a := contextAudit{Reasons: map[string]int{}}
	out := []syntaxContext{}
	bs, e := blocks(raw)
	if e != nil {
		return nil, a, e
	}
	if !strings.EqualFold(language, "Go") {
		a.Reasons["unsupported_language"] = len(bs)
		return out, a, nil
	}
	bt, bok := parseGo(before)
	at, aok := parseGo(after)
	if bt != nil {
		defer bt.Close()
	}
	if at != nil {
		defer at.Close()
	}
	a.BeforeParseOK = bok
	a.AfterParseOK = aok
	seen := map[syntaxContext]bool{}
	for _, b := range bs {
		if (len(b.old) > 0 && !bok) || (len(b.new) > 0 && !aok) {
			a.Reasons["parse_error"]++
			continue
		}
		get := func(t *treesitter.Tree, s string, ls []int) map[string]bool {
			m := map[string]bool{}
			for _, l := range ls {
				n, r := owner(t, s, l)
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
			entry := syntaxContext{b.hunk, n}
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
