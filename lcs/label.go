package lcs

// For each D, vec[D] has length D+1,
// and the label for (D, k) is stored in vec[D][(D+k)/2].
type label struct {
	vec [][]int
}

func (t *label) set(D, k, x int) {
	for len(t.vec) <= D {
		t.vec = append(t.vec, nil)
	}
	if t.vec[D] == nil {
		t.vec[D] = make([]int, D+1)
	}
	t.vec[D][(D+k)/2] = x // known that D+k is even
}

func (t *label) get(d, k int) int {
	return int(t.vec[d][(d+k)/2])
}

func newtriang(limit int) label {
	if limit < 100 {
		// Preallocate if limit is not large.
		return label{vec: make([][]int, limit)}
	}
	return label{}
}
