package diff

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/glaslos/diff/lcs"
)

// An Edit describes the replacement of a portion of a text file.
type Edit struct {
	Start, End int    // byte offsets of the region to replace
	New        string // the replacement
}

// Apply applies a sequence of edits to the src buffer and returns the
// result. Edits are applied in order of start offset; edits with the
// same start offset are applied in they order they were provided.
//
// Apply returns an error if any edit is out of bounds,
// or if any pair of edits is overlapping.
func Apply(src string, edits []Edit) (string, error) {
	edits, size, err := validate(src, edits)
	if err != nil {
		return "", err
	}

	// Apply edits.
	out := make([]byte, 0, size)
	lastEnd := 0
	for _, edit := range edits {
		if lastEnd < edit.Start {
			out = append(out, src[lastEnd:edit.Start]...)
		}
		out = append(out, edit.New...)
		lastEnd = edit.End
	}
	out = append(out, src[lastEnd:]...)

	if len(out) != size {
		panic("wrong size")
	}

	return string(out), nil
}

// Strings computes the differences between two strings.
// The resulting edits respect rune boundaries.
func Strings(before, after string) []Edit {
	if before == after {
		return nil // common case
	}

	if isASCII(before) && isASCII(after) {
		// TODO(adonovan): opt: specialize diffASCII for strings.
		return diffASCII([]byte(before), []byte(after))
	}
	return diffRunes([]rune(before), []rune(after))
}

func diffASCII(before, after []byte) []Edit {
	diffs := lcs.DiffBytes(before, after)

	// Convert from LCS diffs.
	res := make([]Edit, len(diffs))
	for i, d := range diffs {
		res[i] = Edit{d.Start, d.End, string(after[d.ReplStart:d.ReplEnd])}
	}
	return res
}

func diffRunes(before, after []rune) []Edit {
	diffs := lcs.DiffRunes(before, after)

	// The diffs returned by the lcs package use indexes
	// into whatever slice was passed in.
	// Convert rune offsets to byte offsets.
	res := make([]Edit, len(diffs))
	lastEnd := 0
	utf8Len := 0
	for i, d := range diffs {
		utf8Len += runesLen(before[lastEnd:d.Start]) // text between edits
		start := utf8Len
		utf8Len += runesLen(before[d.Start:d.End]) // text deleted by this edit
		res[i] = Edit{start, utf8Len, string(after[d.ReplStart:d.ReplEnd])}
		lastEnd = d.End
	}
	return res
}

// isASCII reports whether s contains only ASCII.
func isASCII[S string | []byte](s S) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// runesLen returns the length in bytes of the UTF-8 encoding of runes.
func runesLen(runes []rune) (len int) {
	for _, r := range runes {
		len += utf8.RuneLen(r)
	}
	return len
}

type editsSort []Edit

// SortEdits orders a slice of Edits by (start, end) offset.
// This ordering puts insertions (end = start) before deletions
// (end > start) at the same point, but uses a stable sort to preserve
// the order of multiple insertions at the same point.
// (Apply detects multiple deletions at the same point as an error.)
func SortEdits(edits []Edit) {
	sort.Stable(editsSort(edits))
}

func (a editsSort) Len() int { return len(a) }
func (a editsSort) Less(i, j int) bool {
	if cmp := a[i].Start - a[j].Start; cmp != 0 {
		return cmp < 0
	}
	return a[i].End < a[j].End
}
func (a editsSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// validate checks that edits are consistent with src,
// and returns the size of the patched output.
// It may return a different slice.
func validate(src string, edits []Edit) ([]Edit, int, error) {
	if !sort.IsSorted(editsSort(edits)) {
		edits = append([]Edit(nil), edits...)
		SortEdits(edits)
	}

	// Check validity of edits and compute final size.
	size := len(src)
	lastEnd := 0
	for _, edit := range edits {
		if !(0 <= edit.Start && edit.Start <= edit.End && edit.End <= len(src)) {
			return nil, 0, fmt.Errorf("diff has out-of-bounds edits")
		}
		if edit.Start < lastEnd {
			return nil, 0, fmt.Errorf("diff has overlapping edits")
		}
		size += len(edit.New) + edit.Start - edit.End
		lastEnd = edit.End
	}

	return edits, size, nil
}
