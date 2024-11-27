package diff

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWordEdits(t *testing.T) {
	tests := []struct {
		name          string
		before, after string
		edits         []Edit
		wordEdits     []Edit
	}{
		{"single", "a\nb\nc\n", "a\nd\nc\n", []Edit{{2, 3, "d"}}, []Edit{{2, 4, "d\n"}}},
		{"end", "a\nb\nc\n", "a\nb\nc\nd\n", []Edit{{6, 6, "d\n"}}, []Edit{{6, 6, "d\n"}}},
		{"add", "a\nb\nc\n", "a\nb\nc\nd", []Edit{{6, 6, "d"}}, []Edit{{6, 6, "d"}}},
		{"rep, add", "a\nb\nc\n", "a\nd\nc\nd", []Edit{{Start: 2, End: 3, New: "d"}, {Start: 6, End: 6, New: "d"}}, []Edit{{2, 6, "d\nc\nd"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			edits := Strings(test.before, test.after)
			require.Equal(t, test.edits, edits)
			edits, err := wordEdits(test.before, edits)
			require.NoError(t, err)
			require.Equal(t, test.wordEdits, edits)
		})
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		content string
		words   []string
	}{
		{"a b c", []string{"a", "b", "c"}},
		{"a\nb\nc", []string{"a", "b", "c"}},
	}

	for _, test := range tests {
		require.Equal(t, test.words, splitWords(test.content))
	}
}

var f = func(s string, delete bool) string {
	if delete {
		return fmt.Sprintf(`<span style="background-color=red">%s</span>`, s)
	}
	return fmt.Sprintf(`<span style="background-color=green">%s</span>`, s)
}

func TestUnifiedFunc(t *testing.T) {
	tests := []struct {
		before, after, expect string
	}{
		{
			`The red fox jumped over the red palace garden fence`,
			`The red fox jumped over the green palace garden fence`,
			`The red fox jumped over the ` + f("red", true) + f("green", false) + ` palace garden fence`,
		},
		{
			`The red fox jumped`,
			`The blue fox fell`,
			`The ` + f("red", true) + f("blue", false) + ` fox ` + f("jumped", true) + f("fell", false),
		},
	}

	for _, test := range tests {
		edits := Strings(test.before, test.after)
		t.Log(edits)

		unified, err := Unified(test.before, edits, f)
		if err != nil {
			t.Fatalf("Unified failed: %v", err)
		}
		require.Equal(t, test.expect, unified)
	}
}
