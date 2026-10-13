package syntax

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestContextBeforeAfterAndAmbiguity(t *testing.T) {
	tests := []struct {
		name, b, a, raw string
		want            []MinimalContext
	}{
		{"replacement", "package p\nfunc F() int {\n return 1\n}\n", "package p\nfunc F() int {\n return 2\n}\n", "@@ -3 +3 @@\n- return 1\n+ return 2\n", []MinimalContext{{0, "F"}}},
		{"deleted file", "package p\nfunc F() int {\n return 1\n}\n", "", "@@ -1,4 +0,0 @@\n-package p\n-func F() int {\n- return 1\n-}\n", []MinimalContext{{0, "F"}}},
		{"new file", "", "package p\nfunc F() int {\n return 2\n}\n", "@@ -0,0 +1,4 @@\n+package p\n+func F() int {\n+ return 2\n+}\n", []MinimalContext{{0, "F"}}},
		{"deleted statement", "package p\nfunc F() {\n println(1)\n}\n", "package p\nfunc F() {\n}\n", "@@ -3 +2,0 @@\n- println(1)\n", []MinimalContext{{0, "F"}}},
		{"rename mismatch", "package p\nfunc F() {}\n", "package p\nfunc G() {}\n", "@@ -2 +2 @@\n-func F() {}\n+func G() {}\n", []MinimalContext{}},
		{"top level", "package p\nvar V=1\n", "package p\nvar V=2\n", "@@ -2 +2 @@\n-var V=1\n+var V=2\n", []MinimalContext{}},
		{"two declarations same line", "package p\nfunc F(){};func G(){}\n", "package p\nfunc F(){println(1)};func G(){println(2)}\n", "@@ -2 +2 @@\n-func F(){};func G(){}\n+func F(){println(1)};func G(){println(2)}\n", []MinimalContext{}},
		{"syntax error", "package p\nfunc F() {}\n", "package p\nfunc F(\n", "@@ -2 +2 @@\n-func F() {}\n+func F(\n", []MinimalContext{}},
		{"multiple hunks", "package p\nfunc F() int {\n return 1\n}\nfunc G() int {\n return 1\n}\n", "package p\nfunc F() int {\n return 2\n}\nfunc G() int {\n return 2\n}\n", "@@ -3 +3 @@\n- return 1\n+ return 2\n@@ -6 +6 @@\n- return 1\n+ return 2\n", []MinimalContext{{0, "F"}, {1, "G"}}},
		{"multiple declarations one hunk", "package p\nfunc F() int {return 1}\nfunc G() int {return 1}\n", "package p\nfunc F() int {return 2}\nfunc G() int {return 2}\n", "@@ -2,2 +2,2 @@\n-func F() int {return 1}\n-func G() int {return 1}\n+func F() int {return 2}\n+func G() int {return 2}\n", []MinimalContext{{0, "F"}, {0, "G"}}},
		{"dedup", "package p\nfunc F() {\n println(1)\n println(1)\n}\n", "package p\nfunc F() {\n println(2)\n println(2)\n}\n", "@@ -3,2 +3,2 @@\n- println(1)\n- println(1)\n+ println(2)\n+ println(2)\n", []MinimalContext{{0, "F"}}},
		{"receiver", "package p\ntype T int\nfunc (x *T) F() int {return 1}\n", "package p\ntype T int\nfunc (x *T) F() int {return 2}\n", "@@ -3 +3 @@\n-func (x *T) F() int {return 1}\n+func (x *T) F() int {return 2}\n", []MinimalContext{{0, "T.F"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, e := extractMinimalContext(context.Background(), tt.b, tt.a, tt.raw, "Go")
			if e != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%v %v want %v", got, e, tt.want)
			}
			again, _, _ := extractMinimalContext(context.Background(), tt.b, tt.a, tt.raw, "Go")
			if !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic")
			}
		})
	}
	if _, _, e := extractMinimalContext(context.Background(), "a", "b", "@@ -1,2 +1 @@\n-a\n+b\n", "Go"); e == nil {
		t.Fatal("malformed diff accepted")
	}
	got, a, e := extractMinimalContext(context.Background(), "a", "b", "@@ -1 +1 @@\n-a\n+b\n", "YAML")
	if e != nil || len(got) != 0 || a.Reasons["unsupported_language"] != 1 {
		t.Fatal("unsupported was guessed")
	}
	got, a, e = extractMinimalContext(context.Background(), "", "package p\nfunc "+strings.Repeat("A", 129)+"() {}\n", "@@ -0,0 +1,2 @@\n+package p\n+func "+strings.Repeat("A", 129)+"() {}\n", "Go")
	if e != nil || len(got) != 0 || a.Reasons["empty_or_long_name"] == 0 {
		t.Fatal("long name not omitted")
	}
}

func TestHunkMetadataCountsEmbeddedHeaders(t *testing.T) {
	raw := "diff --git a/f.go b/f.go\n@@ -1 +1 @@\n-a\n+b\n@@ -3 +3 @@\n-c\n+d\n"
	if len(contextHunkHeader.FindAllStringSubmatch(raw, -1)) != 2 {
		t.Fatal("embedded hunk headers were not counted")
	}
}

func TestContextCapIsSortedAndAudited(t *testing.T) {
	source := "package p\n"
	raw := "@@ -0,0 +1,36 @@\n+package p\n"
	for i := 34; i >= 0; i-- {
		line := fmt.Sprintf("func F%02d() {}\n", i)
		source += line
		raw += "+" + line
	}
	got, a, e := extractMinimalContext(context.Background(), "", source, raw, "Go")
	if e != nil || len(got) != 32 || a.Reasons["entry_cap"] != 3 || got[0].EnclosingDeclaration != "F00" || got[31].EnclosingDeclaration != "F31" {
		t.Fatalf("cap order/audit = %v %v %v", got, a, e)
	}
}

func TestMinimalContextOmitsMultipleDiffSections(t *testing.T) {
	before := "package x\nfunc A() int { return 1 }\n"
	after := "package y\nfunc B() int { return 2 }\n"
	raw := "diff --git a/new.go b/new.go\nnew file mode 100644\n--- /dev/null\n+++ b/new.go\n@@ -0,0 +1,2 @@\n+package y\n+func B() int { return 2 }\ndiff --git a/old.go b/old.go\ndeleted file mode 100644\n--- a/old.go\n+++ /dev/null\n@@ -1,2 +0,0 @@\n-package x\n-func A() int { return 1 }\n"
	got, err := ExtractMinimalContext(context.Background(), []byte(before), []byte(after), raw, "Go")
	if err != nil || len(got) != 0 {
		t.Fatalf("multi-section context should be omitted without stopping raw grouping: context=%v error=%v", got, err)
	}
}
func TestMinimalContextUsesReceiverBaseTypes(t *testing.T) {
	for _, receiver := range []string{"T", "*T", "Box[T]", "*Box[T]"} {
		t.Run(receiver, func(t *testing.T) {
			prefix := "package p\ntype T int\ntype Box[T any] struct{}\n"
			before := prefix + "func (b " + receiver + ") Get() int { return 1 }\n"
			after := prefix + "func (b " + receiver + ") Get() int { return 2 }\n"
			raw := "@@ -4 +4 @@\n-func (b " + receiver + ") Get() int { return 1 }\n+func (b " + receiver + ") Get() int { return 2 }\n"
			want := "T.Get"
			if strings.Contains(receiver, "Box") {
				want = "Box.Get"
			}
			got, err := ExtractMinimalContext(context.Background(), []byte(before), []byte(after), raw, "Go")
			if err != nil || len(got) != 1 || got[0].EnclosingDeclaration != want {
				t.Fatalf("got=%v error=%v want=%s", got, err, want)
			}
		})
	}
}

func TestMinimalContextKeepsCancellationAndMalformedDiffErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw := "diff --git a/new.go b/new.go\ndiff --git a/old.go b/old.go\n"
	if _, err := ExtractMinimalContext(ctx, nil, nil, raw, "Go"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if _, err := ExtractMinimalContext(context.Background(), nil, nil, "@@ -1,2 +1 @@\n-a\n+b\n", "Go"); err == nil {
		t.Fatal("malformed single-section diff was ignored")
	}
}
