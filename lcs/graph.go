package lcs

import "fmt"

// editGraph carries the information for computing the lcs of two sequences.
type editGraph struct {
	seqs   sequences
	vf, vb label // forward and backward labels

	limit int // maximal value of D
	// the bounding rectangle of the current edit graph
	lx, ly, ux, uy int
	delta          int // common subexpression: (ux-lx)-(uy-ly)
}

func (e *editGraph) setForward(d, k, relx int) {
	x := e.lookForward(k, relx)
	e.vf.set(d, k, x-e.lx)
}

// start at (x,y), go up the diagonal as far as possible,
// and label the result with d
func (e *editGraph) lookForward(k, relx int) int {
	rely := relx - k
	x, y := relx+e.lx, rely+e.ly
	if x < e.ux && y < e.uy {
		x += e.seqs.commonPrefixLen(x, e.ux, y, e.uy)
	}
	return x
}

// start at (x,y), go down the diagonal as far as possible,
func (e *editGraph) lookBackward(k, relx int) int {
	rely := relx - (k + e.delta) // forward k = k + e.delta
	x, y := relx+e.lx, rely+e.ly
	if x > 0 && y > 0 {
		x -= e.seqs.commonSuffixLen(0, x, 0, y)
	}
	return x
}

// convert to rectangle, and label the result with d
func (e *editGraph) setBackward(d, k, relx int) {
	x := e.lookBackward(k, relx)
	e.vb.set(d, k, x-e.lx)
}

func (e *editGraph) getForward(d, k int) int {
	x := e.vf.get(d, k)
	return x
}

func (e *editGraph) getBackward(d, k int) int {
	x := e.vb.get(d, k)
	return x
}

// Does Myers' Lemma apply?
func (e *editGraph) twoDone(df, db int) (int, bool) {
	if (df+db+e.delta)%2 != 0 {
		return 0, false // diagonals cannot overlap
	}
	kmin := -db + e.delta
	if -df > kmin {
		kmin = -df
	}
	kmax := db + e.delta
	if df < kmax {
		kmax = df
	}
	for k := kmin; k <= kmax; k += 2 {
		x := e.vf.get(df, k)
		u := e.vb.get(db, k-e.delta)
		if u <= x {
			// is it worth looking at all the other k?
			for l := k; l <= kmax; l += 2 {
				x := e.vf.get(df, l)
				y := x - l
				u := e.vb.get(db, l-e.delta)
				v := u - l
				if x == u || u == 0 || v == 0 || y == e.uy || x == e.ux {
					return l, true
				}
			}
			return k, true
		}
	}
	return 0, false
}

// enforce constraint on d, k
func ok(d, k int) bool {
	return d >= 0 && -d <= k && k <= d
}

// recover the lcs by backtracking from the farthest point reached
func (e *editGraph) forwardlcs(D, k int) lcs {
	var ans lcs
	for x := e.getForward(D, k); x != 0 || x-k != 0; {
		if ok(D-1, k-1) && x-1 == e.getForward(D-1, k-1) {
			// if (x-1,y) is labelled D-1, x--,D--,k--,continue
			D, k, x = D-1, k-1, x-1
			continue
		} else if ok(D-1, k+1) && x == e.getForward(D-1, k+1) {
			// if (x,y-1) is labelled D-1, x, D--,k++, continue
			D, k = D-1, k+1
			continue
		}
		// if (x-1,y-1)--(x,y) is a diagonal, prepend,x--,y--, continue
		y := x - k
		ans = ans.prepend(x+e.lx-1, y+e.ly-1)
		x--
	}
	return ans
}

// recover the lcs by backtracking
func (e *editGraph) backwardlcs(D, k int) lcs {
	var ans lcs
	for x := e.getBackward(D, k); x != e.ux || x-(k+e.delta) != e.uy; {
		if ok(D-1, k-1) && x == e.getBackward(D-1, k-1) {
			// D--, k--, x unchanged
			D, k = D-1, k-1
			continue
		} else if ok(D-1, k+1) && x+1 == e.getBackward(D-1, k+1) {
			// D--, k++, x++
			D, k, x = D-1, k+1, x+1
			continue
		}
		y := x - (k + e.delta)
		ans = ans.append(x+e.lx, y+e.ly)
		x++
	}
	return ans
}

func (e *editGraph) twolcs(df, db, kf int) lcs {
	// db==df || db+1==df
	x := e.vf.get(df, kf)
	y := x - kf
	kb := kf - e.delta
	u := e.vb.get(db, kb)
	v := u - kf

	// Myers proved there is a df-path from (0,0) to (u,v)
	// and a db-path from (x,y) to (N,M).
	// In the first case the overall path is the forward path
	// to (u,v) followed by the backward path to (N,M).
	// In the second case the path is the backward path to (x,y)
	// followed by the forward path to (x,y) from (0,0).

	// Look for some special cases to avoid computing either of these paths.
	if x == u {
		// "babaab" "cccaba"
		// already patched together
		lcs := e.forwardlcs(df, kf)
		lcs = append(lcs, e.backwardlcs(db, kb)...)
		return lcs.sort()
	}

	// is (u-1,v) or (u,v-1) labelled df-1?
	// if so, that forward df-1-path plus a horizontal or vertical edge
	// is the df-path to (u,v), then plus the db-path to (N,M)
	if u > 0 && ok(df-1, u-1-v) && e.vf.get(df-1, u-1-v) == u-1 {
		//  "aabbab" "cbcabc"
		lcs := e.forwardlcs(df-1, u-1-v)
		lcs = append(lcs, e.backwardlcs(db, kb)...)
		return lcs.sort()
	}
	if v > 0 && ok(df-1, (u-(v-1))) && e.vf.get(df-1, u-(v-1)) == u {
		//  "abaabb" "bcacab"
		lcs := e.forwardlcs(df-1, u-(v-1))
		lcs = append(lcs, e.backwardlcs(db, kb)...)
		return lcs.sort()
	}

	// The path can't possibly contribute to the lcs because it
	// is all horizontal or vertical edges
	if u == 0 || v == 0 || x == e.ux || y == e.uy {
		// "abaabb" "abaaaa"
		if u == 0 || v == 0 {
			return e.backwardlcs(db, kb)
		}
		return e.forwardlcs(df, kf)
	}

	// is (x+1,y) or (x,y+1) labelled db-1?
	if x+1 <= e.ux && ok(db-1, x+1-y-e.delta) && e.vb.get(db-1, x+1-y-e.delta) == x+1 {
		// "bababb" "baaabb"
		lcs := e.backwardlcs(db-1, kb+1)
		lcs = append(lcs, e.forwardlcs(df, kf)...)
		return lcs.sort()
	}
	if y+1 <= e.uy && ok(db-1, x-(y+1)-e.delta) && e.vb.get(db-1, x-(y+1)-e.delta) == x {
		// "abbbaa" "cabacc"
		lcs := e.backwardlcs(db-1, kb-1)
		lcs = append(lcs, e.forwardlcs(df, kf)...)
		return lcs.sort()
	}

	// need to compute another path
	// "aabbaa" "aacaba"
	lcs := e.backwardlcs(db, kb)
	oldx, oldy := e.ux, e.uy
	e.ux = u
	e.uy = v
	lcs = append(lcs, forward(e)...)
	e.ux, e.uy = oldx, oldy
	return lcs.sort()
}

// fdone decides if the forward path has reached the upper right
// corner of the rectangle. If so, it also returns the computed lcs.
func (e *editGraph) fdone(D, k int) (bool, lcs) {
	// x, y, k are relative to the rectangle
	x := e.vf.get(D, k)
	y := x - k
	if x == e.ux && y == e.uy {
		return true, e.forwardlcs(D, k)
	}
	return false, nil
}

// bdone decides if the backward path has reached the lower left corner
func (e *editGraph) bdone(D, k int) (bool, lcs) {
	// x, y, k are relative to the rectangle
	x := e.vb.get(D, k)
	y := x - (k + e.delta)
	if x == 0 && y == 0 {
		return true, e.backwardlcs(D, k)
	}
	return false, nil
}

// run the backward algorithm, until success or up to the limit on D.
func backward(e *editGraph) lcs {
	e.setBackward(0, 0, e.ux)
	if ok, ans := e.bdone(0, 0); ok {
		return ans
	}
	// from D to D+1
	for D := 0; D < e.limit; D++ {
		e.setBackward(D+1, -(D + 1), e.getBackward(D, -D)-1)
		if ok, ans := e.bdone(D+1, -(D + 1)); ok {
			return ans
		}
		e.setBackward(D+1, D+1, e.getBackward(D, D))
		if ok, ans := e.bdone(D+1, D+1); ok {
			return ans
		}
		for k := -D + 1; k <= D-1; k += 2 {
			// these are tricky and easy to get wrong
			lookv := e.lookBackward(k, e.getBackward(D, k-1))
			lookh := e.lookBackward(k, e.getBackward(D, k+1)-1)
			if lookv < lookh {
				e.setBackward(D+1, k, lookv)
			} else {
				e.setBackward(D+1, k, lookh)
			}
			if ok, ans := e.bdone(D+1, k); ok {
				return ans
			}
		}
	}

	// D is too large
	// find the D path with minimal x+y inside the rectangle and
	// use that to compute the part of the lcs found
	kmax := -e.limit - 1
	diagmin := 1 << 25
	for k := -e.limit; k <= e.limit; k += 2 {
		x := e.getBackward(e.limit, k)
		y := x - (k + e.delta)
		if x+y < diagmin && x >= 0 && y >= 0 {
			diagmin, kmax = x+y, k
		}
	}
	if kmax < -e.limit {
		panic(fmt.Sprintf("no paths when limit=%d?", e.limit))
	}
	return e.backwardlcs(e.limit, kmax)
}
