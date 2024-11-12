package diff_test

import (
	"testing"

	"github.com/glaslos/diff"
)

const (
	FileA         = "from"
	FileB         = "to"
	UnifiedPrefix = "--- " + FileA + "\n+++ " + FileB + "\n"
)

var TestCases = []struct {
	Name, In, Out, Unified string
	Edits, LineEdits       []diff.Edit // expectation (LineEdits=nil => already line-aligned)
	NoDiff                 bool
}{{
	Name: "empty",
	In:   "",
	Out:  "",
}, {
	Name: "no_diff",
	In:   "gargantuan\n",
	Out:  "gargantuan\n",
}, {
	Name: "replace_all",
	In:   "fruit\n",
	Out:  "cheese\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-fruit
+cheese
`[1:],
	Edits:     []diff.Edit{{Start: 0, End: 5, New: "cheese"}},
	LineEdits: []diff.Edit{{Start: 0, End: 6, New: "cheese\n"}},
}, {
	Name: "insert_rune",
	In:   "gord\n",
	Out:  "gourd\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-gord
+gourd
`[1:],
	Edits:     []diff.Edit{{Start: 2, End: 2, New: "u"}},
	LineEdits: []diff.Edit{{Start: 0, End: 5, New: "gourd\n"}},
}, {
	Name: "delete_rune",
	In:   "groat\n",
	Out:  "goat\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-groat
+goat
`[1:],
	Edits:     []diff.Edit{{Start: 1, End: 2, New: ""}},
	LineEdits: []diff.Edit{{Start: 0, End: 6, New: "goat\n"}},
}, {
	Name: "replace_rune",
	In:   "loud\n",
	Out:  "lord\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-loud
+lord
`[1:],
	Edits:     []diff.Edit{{Start: 2, End: 3, New: "r"}},
	LineEdits: []diff.Edit{{Start: 0, End: 5, New: "lord\n"}},
}, {
	Name: "replace_partials",
	In:   "blanket\n",
	Out:  "bunker\n",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-blanket
+bunker
`[1:],
	Edits: []diff.Edit{
		{Start: 1, End: 3, New: "u"},
		{Start: 6, End: 7, New: "r"},
	},
	LineEdits: []diff.Edit{{Start: 0, End: 8, New: "bunker\n"}},
}, {
	Name: "insert_line",
	In:   "1: one\n3: three\n",
	Out:  "1: one\n2: two\n3: three\n",
	Unified: UnifiedPrefix + `
@@ -1,2 +1,3 @@
 1: one
+2: two
 3: three
`[1:],
	Edits: []diff.Edit{{Start: 7, End: 7, New: "2: two\n"}},
}, {
	Name: "replace_no_newline",
	In:   "A",
	Out:  "B",
	Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+B
\ No newline at end of file
`[1:],
	Edits: []diff.Edit{{Start: 0, End: 1, New: "B"}},
}, {
	Name: "delete_empty",
	In:   "meow",
	Out:  "", // GNU diff -u special case: +0,0
	Unified: UnifiedPrefix + `
@@ -1 +0,0 @@
-meow
\ No newline at end of file
`[1:],
	Edits:     []diff.Edit{{Start: 0, End: 4, New: ""}},
	LineEdits: []diff.Edit{{Start: 0, End: 4, New: ""}},
}, {
	Name: "append_empty",
	In:   "", // GNU diff -u special case: -0,0
	Out:  "AB\nC",
	Unified: UnifiedPrefix + `
@@ -0,0 +1,2 @@
+AB
+C
\ No newline at end of file
`[1:],
	Edits:     []diff.Edit{{Start: 0, End: 0, New: "AB\nC"}},
	LineEdits: []diff.Edit{{Start: 0, End: 0, New: "AB\nC"}},
},
	// TODO(adonovan): fix this test: GNU diff -u prints "+1,2", Unifies prints "+1,3".
	// 	{
	// 		Name: "add_start",
	// 		In:   "A",
	// 		Out:  "B\nCA",
	// 		Unified: UnifiedPrefix + `
	// @@ -1 +1,2 @@
	// -A
	// \ No newline at end of file
	// +B
	// +CA
	// \ No newline at end of file
	// `[1:],
	// 		Edits:     []diff.TextEdit{{Span: newSpan(0, 0), NewText: "B\nC"}},
	// 		LineEdits: []diff.TextEdit{{Span: newSpan(0, 0), NewText: "B\nC"}},
	// 	},
	{
		Name: "add_end",
		In:   "A",
		Out:  "AB",
		Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+AB
\ No newline at end of file
`[1:],
		Edits:     []diff.Edit{{Start: 1, End: 1, New: "B"}},
		LineEdits: []diff.Edit{{Start: 0, End: 1, New: "AB"}},
	}, {
		Name: "add_empty",
		In:   "",
		Out:  "AB\nC",
		Unified: UnifiedPrefix + `
@@ -0,0 +1,2 @@
+AB
+C
\ No newline at end of file
`[1:],
		Edits:     []diff.Edit{{Start: 0, End: 0, New: "AB\nC"}},
		LineEdits: []diff.Edit{{Start: 0, End: 0, New: "AB\nC"}},
	}, {
		Name: "add_newline",
		In:   "A",
		Out:  "A\n",
		Unified: UnifiedPrefix + `
@@ -1 +1 @@
-A
\ No newline at end of file
+A
`[1:],
		Edits:     []diff.Edit{{Start: 1, End: 1, New: "\n"}},
		LineEdits: []diff.Edit{{Start: 0, End: 1, New: "A\n"}},
	}, {
		Name: "delete_front",
		In:   "A\nB\nC\nA\nB\nB\nA\n",
		Out:  "C\nB\nA\nB\nA\nC\n",
		Unified: UnifiedPrefix + `
@@ -1,7 +1,6 @@
-A
-B
 C
+B
 A
 B
-B
 A
+C
`[1:],
		NoDiff: true, // unified diff is different but valid
		Edits: []diff.Edit{
			{Start: 0, End: 4, New: ""},
			{Start: 6, End: 6, New: "B\n"},
			{Start: 10, End: 12, New: ""},
			{Start: 14, End: 14, New: "C\n"},
		},
		LineEdits: []diff.Edit{
			{Start: 0, End: 4, New: ""},
			{Start: 6, End: 6, New: "B\n"},
			{Start: 10, End: 12, New: ""},
			{Start: 14, End: 14, New: "C\n"},
		},
	}, {
		Name: "replace_last_line",
		In:   "A\nB\n",
		Out:  "A\nC\n\n",
		Unified: UnifiedPrefix + `
@@ -1,2 +1,3 @@
 A
-B
+C
+
`[1:],
		Edits:     []diff.Edit{{Start: 2, End: 3, New: "C\n"}},
		LineEdits: []diff.Edit{{Start: 2, End: 4, New: "C\n\n"}},
	},
	{
		Name: "multiple_replace",
		In:   "A\nB\nC\nD\nE\nF\nG\n",
		Out:  "A\nH\nI\nJ\nE\nF\nK\n",
		Unified: UnifiedPrefix + `
@@ -1,7 +1,7 @@
 A
-B
-C
-D
+H
+I
+J
 E
 F
-G
+K
`[1:],
		Edits: []diff.Edit{
			{Start: 2, End: 8, New: "H\nI\nJ\n"},
			{Start: 12, End: 14, New: "K\n"},
		},
		NoDiff: true, // diff algorithm produces different delete/insert pattern
	},
	{
		Name:  "extra_newline",
		In:    "\nA\n",
		Out:   "A\n",
		Edits: []diff.Edit{{Start: 0, End: 1, New: ""}},
		Unified: UnifiedPrefix + `@@ -1,2 +1 @@
-
 A
`,
	}, {
		Name:      "unified_lines",
		In:        "aaa\nccc\n",
		Out:       "aaa\nbbb\nccc\n",
		Edits:     []diff.Edit{{Start: 3, End: 3, New: "\nbbb"}},
		LineEdits: []diff.Edit{{Start: 0, End: 4, New: "aaa\nbbb\n"}},
		Unified:   UnifiedPrefix + "@@ -1,2 +1,3 @@\n aaa\n+bbb\n ccc\n",
	}, {
		Name: "60379",
		In: `package a

type S struct {
s fmt.Stringer
}
`,
		Out: `package a

type S struct {
	s fmt.Stringer
}
`,
		Edits:     []diff.Edit{{Start: 27, End: 27, New: "\t"}},
		LineEdits: []diff.Edit{{Start: 27, End: 42, New: "\ts fmt.Stringer\n"}},
		Unified:   UnifiedPrefix + "@@ -1,5 +1,5 @@\n package a\n \n type S struct {\n-s fmt.Stringer\n+\ts fmt.Stringer\n }\n",
	},
}

func TestNEdits(t *testing.T) {
	for _, tc := range TestCases {
		edits := diff.Strings(tc.In, tc.Out)
		got, err := diff.Apply(tc.In, edits)
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}
		if got != tc.Out {
			t.Fatalf("%s: got %q wanted %q", tc.Name, got, tc.Out)
		}
		if len(edits) < len(tc.Edits) { // should find subline edits
			t.Errorf("got %v, expected %v for %#v", edits, tc.Edits, tc)
		}
	}
}

func TestUnifiedFunc(t *testing.T) {
	edits := diff.Strings("a\nb\nc\n", "a\nd\nc\n")
	unified, err := diff.UnifiedFn("a", "b", "c", edits, 1, func(s string) string { return "" })
	if err != nil {
		t.Fatalf("Unified failed: %v", err)
	}
	if unified != "@@ -1,3 +1,3 @@\n a\n-b\n+d\n c\n" {
		t.Fatalf("got %q", unified)
	}
}
