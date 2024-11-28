package diff

import (
	"strings"
)

func Unified(
	content string, edits []Edit,
	split func(string) []string,
	format func(string, bool) string,
) (string, error) {
	u, err := toUnified(content, edits, split)
	if err != nil {
		return "", err
	}
	return u.String(format), nil
}

// opKind is used to denote the type of operation a line represents.
type opKind int

const (
	// opDelete is the operation kind for a line that is present in the input
	// but not in the output.
	opDelete opKind = iota
	// opInsert is the operation kind for a line that is new in the output.
	opInsert
	// opEqual is the operation kind for a line that is the same in the input and
	// output, often used to provide context around edited lines.
	opEqual
)

// unified represents a set of edits as a unified diff.
type unified struct {
	words []word
}

// word represents a single word operation to apply as part of a Hunk.
type word struct {
	// kind is the type of word this represents, deletion, insertion or copy.
	kind opKind
	// content is the content of this word.
	// For deletion it is the word being removed, for all others it is the word
	// to put in the output.
	content string
}

// toUnified takes a file contents and a sequence of edits, and calculates
// a unified diff that represents those edits.
func toUnified(
	content string, edits []Edit,
	split func(string) []string,
) (*unified, error) {
	if len(edits) == 0 {
		return nil, nil
	}
	var err error
	edits, err = wordEdits(content, edits) // expand to whole words
	if err != nil {
		return nil, err
	}
	words := split(content)

	u := &unified{
		words: make([]word, 0, len(words)),
	}

	previous := 0
	toWord := 0
	for _, edit := range edits {
		// Compute the zero-based line numbers of the edit start and end.
		// TODO(adonovan): opt: compute incrementally, avoid O(n^2).
		start := strings.Count(content[:edit.Start], " ")
		end := strings.Count(content[:edit.End], " ")
		if edit.End == len(content) && len(content) > 0 && content[len(content)-1] != ' ' {
			end++ // EOF counts as an implicit newline
		}

		// add all leading words
		if previous == 0 {
			addEqualWords(u, words, previous, start)
		}

		switch {
		case previous != 0 && start == previous:
			//direct extension
		case previous != 0 && start <= previous+2:
			//within range of previous lines, add the joiners
			addEqualWords(u, words, previous, start)
		default:
			//need to start a new hunk
			if previous != 0 {
				// add the edge to the previous hunk
				addEqualWords(u, words, previous, previous+2)
				//u.hunks = append(u.hunks, h)
			}
			toWord += start - previous
			// add the edge to the new hunk
			//delta := addEqualLines(u, words, start-2, start)
			//h.fromWord -= delta
			//h.toWord -= delta
		}
		previous = start
		for i := start; i < end; i++ {
			u.words = append(u.words, word{kind: opDelete, content: words[i]})
			previous++
		}
		if edit.New != "" {
			for _, content := range split(edit.New) {
				u.words = append(u.words, word{kind: opInsert, content: content})
				toWord++
			}
		}
	}
	if previous != 0 {
		// add the edge to the final hunk
		addEqualWords(u, words, previous, len(words))
		//u.words = append(u.words, h)
	}
	return u, nil
}

func addEqualWords(u *unified, words []string, start, end int) int {
	delta := 0
	for i := start; i < end; i++ {
		if i < 0 {
			continue
		}
		if i >= len(words) {
			return delta
		}
		u.words = append(u.words, word{kind: opEqual, content: words[i]})
		delta++
	}
	return delta
}

// wordEdits expands and merges a sequence of edits so that each
// resulting edit replaces one or more complete word.
// See ApplyEdits for preconditions.
func wordEdits(src string, edits []Edit) ([]Edit, error) {
	edits, _, err := validate(src, edits)
	if err != nil {
		return nil, err
	}

	// Do all deletions begin and end at the start of a word
	// (This is merely a fast path.)
	for _, edit := range edits {
		if edit.Start >= len(src) || // insertion at EOF
			edit.Start > 0 && src[edit.Start-1] != ' ' || // not at line start
			edit.End > 0 && src[edit.End-1] != ' ' || // not at line start
			edit.New != "" && edit.New[len(edit.New)-1] != ' ' { // partial insert
			goto expand // slow path
		}
	}
	return edits, nil // aligned

expand:
	if len(edits) == 0 {
		return edits, nil // no edits (unreachable due to fast path)
	}
	expanded := make([]Edit, 0, len(edits)) // a guess
	prev := edits[0]
	// TODO(adonovan): opt: start from the first misaligned edit.
	// TODO(adonovan): opt: avoid quadratic cost of string += string.
	for _, edit := range edits[1:] {
		between := src[prev.End:edit.Start]
		if !strings.Contains(between, " ") {
			// overlapping words: combine with previous edit.
			prev.New += between + edit.New
			prev.End = edit.End
		} else {
			// non-overlapping words: flush previous edit.
			expanded = append(expanded, expandEdit(prev, src))
			prev = edit
		}
	}
	return append(expanded, expandEdit(prev, src)), nil // flush final edit
}

// expandEdit returns edit expanded to complete whole words.
func expandEdit(edit Edit, src string) Edit {
	// Expand start left to start of line.
	// (delta is the zero-based column number of start.)
	start := edit.Start
	if delta := start - 1 - strings.LastIndex(src[:start], " "); delta > 0 {
		edit.Start -= delta
		edit.New = src[start-delta:start] + edit.New
	}

	// Expand end right to end of line.
	end := edit.End
	if end > 0 && src[end-1] != ' ' ||
		edit.New != "" && edit.New[len(edit.New)-1] != ' ' {
		if nl := strings.IndexByte(src[end:], ' '); nl < 0 {
			edit.End = len(src) // extend to EOF
		} else {
			edit.End = end + nl + 1 // extend beyond \n
		}
	}
	edit.New += src[end:edit.End]

	return edit
}

// String converts a unified diff to the standard textual form for that diff.
// The output of this function can be passed to tools like patch.
func (u unified) String(format func(content string, delete bool) string) string {
	if len(u.words) == 0 {
		return ""
	}

	s := make([]string, 0, len(u.words))
	for i, l := range u.words {
		switch l.kind {
		case opDelete:
			s = append(s, format(l.content, true))
		case opInsert:
			s = append(s, format(l.content, false))
			if i != len(u.words)-1 {
				s = append(s, " ") // space after all insertions but the last
			}
		default:
			s = append(s, l.content)
			if i != len(u.words)-1 {
				s = append(s, " ") // space after all but the last word
			}
		}
	}
	return strings.Join(s, "")
}
