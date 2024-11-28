package diff

import (
	"fmt"
	"strings"
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
		{"single", "a b c", "a d c", []Edit{{2, 3, "d"}}, []Edit{{2, 4, "d "}}},
		{"add", "a b c", "a b c d", []Edit{{5, 5, " d"}}, []Edit{{4, 5, "c d"}}},
		{"rep, add", "a b c", "a d c d", []Edit{{Start: 2, End: 3, New: "d"}, {Start: 5, End: 5, New: " d"}}, []Edit{{2, 4, "d "}, {4, 5, "c d"}}},
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
		{"a\nb\nc", []string{"a\nb\nc"}},
	}

	for _, test := range tests {
		require.Equal(t, test.words, split(test.content))
	}
}

var format = func(s string, delete bool) string {
	if delete {
		return fmt.Sprintf(`<span style="background-color=red">%s</span>`, s)
	}
	return fmt.Sprintf(`<span style="background-color=green">%s</span>`, s)
}

var split = func(text string) []string {
	words := strings.Split(text, " ")
	if words[len(words)-1] == "" {
		words = words[:len(words)-1]
	}
	return words
}

func TestUnifiedFunc(t *testing.T) {
	tests := []struct {
		before, after, expect string
	}{
		{
			`The red fox jumped over the red palace garden fence`,
			`The red fox jumped over the green palace garden fence`,
			`The red fox jumped over the ` + format("red", true) + format("green", false) + ` palace garden fence`,
		},
		{
			`The red fox jumped`,
			`The blue fox fell`,
			`The ` + format("red", true) + format("blue", false) + ` fox ` + format("jumped", true) + format("fell", false),
		},
		{
			`The red fox jumped 
			over the red palace garden fence`,
			`The red fox fell 
			over the red palace garden fence`,
			`The red fox ` + format("jumped", true) + format("fell", false) + ` 
			over the red palace garden fence`,
		},
	}

	for _, test := range tests {
		edits := Strings(test.before, test.after)
		t.Log(edits)

		unified, err := Unified(test.before, edits, split, format)
		if err != nil {
			t.Fatalf("Unified failed: %v", err)
		}
		require.Equal(t, test.expect, unified)
	}
}
