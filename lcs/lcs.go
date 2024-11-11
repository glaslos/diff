package lcs

import (
	"fmt"
	"log"
	"sort"
)

// lcs is a longest common sequence
type lcs []diag

// toDiffs converts an LCS to a list of edits.
func (l lcs) toDiffs(alen, blen int) []Diff {
	var diffs []Diff
	var pa, pb int // offsets in a, b
	for _, l := range l {
		if pa < l.X || pb < l.Y {
			diffs = append(diffs, Diff{pa, l.X, pb, l.Y})
		}
		pa = l.X + l.Len
		pb = l.Y + l.Len
	}
	if pa < alen || pb < blen {
		diffs = append(diffs, Diff{pa, alen, pb, blen})
	}
	return diffs
}

// prepend a diagonal (x,y)-(x+1,y+1) segment either to an empty lcs
// or to its first Diag. prepend is only called to extend diagonals
// the backward direction.
func (l lcs) prepend(x, y int) lcs {
	if len(l) > 0 {
		d := &l[0]
		if int(d.X) == x+1 && int(d.Y) == y+1 {
			// extend the diagonal down and to the left
			d.X, d.Y = int(x), int(y)
			d.Len++
			return l
		}
	}

	r := diag{X: int(x), Y: int(y), Len: 1}
	l = append([]diag{r}, l...)
	return l
}

// sort sorts in place, by lowest X, and if tied, inversely by Len
func (l lcs) sort() lcs {
	sort.Slice(l, func(i, j int) bool {
		if l[i].X != l[j].X {
			return l[i].X < l[j].X
		}
		return l[i].Len > l[j].Len
	})
	return l
}

// validate that the elements of the lcs do not overlap
// (can only happen when the two-sided algorithm ends early)
// expects the lcs to be sorted
func (l lcs) valid() bool {
	for i := 1; i < len(l); i++ {
		if l[i-1].X+l[i-1].Len > l[i].X {
			return false
		}
		if l[i-1].Y+l[i-1].Len > l[i].Y {
			return false
		}
	}
	return true
}

// append appends a diagonal, or extends the existing one.
// by adding the edge (x,y)-(x+1.y+1). append is only called
// to extend diagonals in the forward direction.
func (l lcs) append(x, y int) lcs {
	if len(l) > 0 {
		last := &l[len(l)-1]
		// Expand last element if adjoining.
		if last.X+last.Len == x && last.Y+last.Len == y {
			last.Len++
			return l
		}
	}

	return append(l, diag{X: x, Y: y, Len: 1})
}

// repair overlapping lcs
// only called if two-sided stops early
func (l lcs) fix() lcs {
	// from the set of diagonals in l, find a maximal non-conflicting set
	// this problem may be NP-complete, but we use a greedy heuristic,
	// which is quadratic, but with a better data structure, could be D log D.
	// indepedent is not enough: {0,3,1} and {3,0,2} can't both occur in an lcs
	// which has to have monotone x and y
	if len(l) == 0 {
		return nil
	}
	sort.Slice(l, func(i, j int) bool { return l[i].Len > l[j].Len })
	tmp := make(lcs, 0, len(l))
	tmp = append(tmp, l[0])
	for i := 1; i < len(l); i++ {
		var dir direction
		nxt := l[i]
		for _, in := range tmp {
			if dir, nxt = overlap(in, nxt); dir == empty || dir == bad {
				break
			}
		}
		if nxt.Len > 0 && dir != bad {
			tmp = append(tmp, nxt)
		}
	}
	tmp.sort()
	if false && !tmp.valid() { // debug checking
		log.Fatalf("here %d", len(tmp))
	}
	return tmp
}

// A diag is a piece of the edit graph where A[X+i] == B[Y+i], for 0<=i<Len.
// All computed diagonals are parts of a longest common subsequence.
type diag struct {
	X, Y int
	Len  int
}

// A Diff is a replacement of a portion of A by a portion of B.
type Diff struct {
	Start, End         int // offsets of portion to delete in A
	ReplStart, ReplEnd int // offset of replacement text in B
}

// DiffBytes returns the differences between two byte sequences.
// It does not respect rune boundaries.
func DiffBytes(a, b []byte) []Diff { return diff(bytesSeqs{a, b}) }

// DiffRunes returns the differences between two rune sequences.
func DiffRunes(a, b []rune) []Diff { return diff(runesSeqs{a, b}) }

func diff(seqs sequences) []Diff {
	// A limit on how deeply the LCS algorithm should search. The value is just a guess.
	const maxDiffs = 100
	diff, _ := compute(seqs, twosided, maxDiffs/2)
	return diff
}

// compute computes the list of differences between two sequences,
// along with the LCS. It is exercised directly by tests.
// The algorithm is one of {forward, backward, twosided}.
func compute(seqs sequences, algo func(*editGraph) lcs, limit int) ([]Diff, lcs) {
	if limit <= 0 {
		limit = 1 << 25 // effectively infinity
	}
	alen, blen := seqs.lengths()
	g := &editGraph{
		seqs:  seqs,
		vf:    newtriang(limit),
		vb:    newtriang(limit),
		limit: limit,
		ux:    alen,
		uy:    blen,
		delta: alen - blen,
	}
	lcs := algo(g)
	diffs := lcs.toDiffs(alen, blen)
	return diffs, lcs
}

func twosided(e *editGraph) lcs {
	// The termination condition could be improved, as either the forward
	// or backward pass could succeed before Myers' Lemma applies.
	// Aside from questions of efficiency (is the extra testing cost-effective)
	// this is more likely to matter when e.limit is reached.
	e.setForward(0, 0, e.lx)
	e.setBackward(0, 0, e.ux)

	// from D to D+1
	for D := 0; D < e.limit; D++ {
		// just finished a backwards pass, so check
		if got, ok := e.twoDone(D, D); ok {
			return e.twolcs(D, D, got)
		}
		// do a forwards pass (D to D+1)
		e.setForward(D+1, -(D + 1), e.getForward(D, -D))
		e.setForward(D+1, D+1, e.getForward(D, D)+1)
		for k := -D + 1; k <= D-1; k += 2 {
			// these are tricky and easy to get backwards
			lookv := e.lookForward(k, e.getForward(D, k-1)+1)
			lookh := e.lookForward(k, e.getForward(D, k+1))
			if lookv > lookh {
				e.setForward(D+1, k, lookv)
			} else {
				e.setForward(D+1, k, lookh)
			}
		}
		// just did a forward pass, so check
		if got, ok := e.twoDone(D+1, D); ok {
			return e.twolcs(D+1, D, got)
		}
		// do a backward pass, D to D+1
		e.setBackward(D+1, -(D + 1), e.getBackward(D, -D)-1)
		e.setBackward(D+1, D+1, e.getBackward(D, D))
		for k := -D + 1; k <= D-1; k += 2 {
			// these are tricky and easy to get wrong
			lookv := e.lookBackward(k, e.getBackward(D, k-1))
			lookh := e.lookBackward(k, e.getBackward(D, k+1)-1)
			if lookv < lookh {
				e.setBackward(D+1, k, lookv)
			} else {
				e.setBackward(D+1, k, lookh)
			}
		}
	}

	// D too large. combine a forward and backward partial lcs
	// first, a forward one
	kmax := -e.limit - 1
	diagmax := -1
	for k := -e.limit; k <= e.limit; k += 2 {
		x := e.getForward(e.limit, k)
		y := x - k
		if x+y > diagmax && x <= e.ux && y <= e.uy {
			diagmax, kmax = x+y, k
		}
	}
	if kmax < -e.limit {
		panic(fmt.Sprintf("no forward paths when limit=%d?", e.limit))
	}
	lcs := e.forwardlcs(e.limit, kmax)
	// now a backward one
	// find the D path with minimal x+y inside the rectangle and
	// use that to compute the lcs
	diagmin := 1 << 25 // infinity
	for k := -e.limit; k <= e.limit; k += 2 {
		x := e.getBackward(e.limit, k)
		y := x - (k + e.delta)
		if x+y < diagmin && x >= 0 && y >= 0 {
			diagmin, kmax = x+y, k
		}
	}
	if kmax < -e.limit {
		panic(fmt.Sprintf("no backward paths when limit=%d?", e.limit))
	}
	lcs = append(lcs, e.backwardlcs(e.limit, kmax)...)
	// These may overlap (e.forwardlcs and e.backwardlcs return sorted lcs)
	ans := lcs.fix()
	return ans
}

type direction int

const (
	empty    direction = iota // diag is empty (so not in lcs)
	leftdown                  // proposed acceptably to the left and below
	rightup                   // proposed diag is acceptably to the right and above
	bad                       // proposed diag is inconsistent with the lcs so far
)

// overlap trims the proposed diag prop  so it doesn't overlap with
// the existing diag that has already been added to the lcs.
func overlap(exist, prop diag) (direction, diag) {
	if prop.X <= exist.X && exist.X < prop.X+prop.Len {
		// remove the end of prop where it overlaps with the X end of exist
		delta := prop.X + prop.Len - exist.X
		prop.Len -= delta
		if prop.Len <= 0 {
			return empty, prop
		}
	}
	if exist.X <= prop.X && prop.X < exist.X+exist.Len {
		// remove the beginning of prop where overlaps with exist
		delta := exist.X + exist.Len - prop.X
		prop.Len -= delta
		if prop.Len <= 0 {
			return empty, prop
		}
		prop.X += delta
		prop.Y += delta
	}
	if prop.Y <= exist.Y && exist.Y < prop.Y+prop.Len {
		// remove the end of prop that overlaps (in Y) with exist
		delta := prop.Y + prop.Len - exist.Y
		prop.Len -= delta
		if prop.Len <= 0 {
			return empty, prop
		}
	}
	if exist.Y <= prop.Y && prop.Y < exist.Y+exist.Len {
		// remove the beginning of peop that overlaps with exist
		delta := exist.Y + exist.Len - prop.Y
		prop.Len -= delta
		if prop.Len <= 0 {
			return empty, prop
		}
		prop.X += delta // no test reaches this code
		prop.Y += delta
	}
	if prop.X+prop.Len <= exist.X && prop.Y+prop.Len <= exist.Y {
		return leftdown, prop
	}
	if exist.X+exist.Len <= prop.X && exist.Y+exist.Len <= prop.Y {
		return rightup, prop
	}
	// prop can't be in an lcs that contains exist
	return bad, prop
}

// run the forward algorithm, until success or up to the limit on D.
func forward(e *editGraph) lcs {
	e.setForward(0, 0, e.lx)
	if ok, ans := e.fdone(0, 0); ok {
		return ans
	}
	// from D to D+1
	for D := 0; D < e.limit; D++ {
		e.setForward(D+1, -(D + 1), e.getForward(D, -D))
		if ok, ans := e.fdone(D+1, -(D + 1)); ok {
			return ans
		}
		e.setForward(D+1, D+1, e.getForward(D, D)+1)
		if ok, ans := e.fdone(D+1, D+1); ok {
			return ans
		}
		for k := -D + 1; k <= D-1; k += 2 {
			// these are tricky and easy to get backwards
			lookv := e.lookForward(k, e.getForward(D, k-1)+1)
			lookh := e.lookForward(k, e.getForward(D, k+1))
			if lookv > lookh {
				e.setForward(D+1, k, lookv)
			} else {
				e.setForward(D+1, k, lookh)
			}
			if ok, ans := e.fdone(D+1, k); ok {
				return ans
			}
		}
	}
	// D is too large
	// find the D path with maximal x+y inside the rectangle and
	// use that to compute the found part of the lcs
	kmax := -e.limit - 1
	diagmax := -1
	for k := -e.limit; k <= e.limit; k += 2 {
		x := e.getForward(e.limit, k)
		y := x - k
		if x+y > diagmax && x <= e.ux && y <= e.uy {
			diagmax, kmax = x+y, k
		}
	}
	return e.forwardlcs(e.limit, kmax)
}
