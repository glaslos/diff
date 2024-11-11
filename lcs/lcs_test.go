// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lcs

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
)

type Btest struct {
	a, b string
	lcs  []string
}

var Btests = []Btest{
	{"aaabab", "abaab", []string{"abab", "aaab"}},
	{"aabbba", "baaba", []string{"aaba"}},
	{"cabbx", "cbabx", []string{"cabx", "cbbx"}},
	{"c", "cb", []string{"c"}},
	{"aaba", "bbb", []string{"b"}},
	{"bbaabb", "b", []string{"b"}},
	{"baaabb", "bbaba", []string{"bbb", "baa", "bab"}},
	{"baaabb", "abbab", []string{"abb", "bab", "aab"}},
	{"baaba", "aaabba", []string{"aaba"}},
	{"ca", "cba", []string{"ca"}},
	{"ccbcbc", "abba", []string{"bb"}},
	{"ccbcbc", "aabba", []string{"bb"}},
	{"ccb", "cba", []string{"cb"}},
	{"caef", "axe", []string{"ae"}},
	{"bbaabb", "baabb", []string{"baabb"}},
	// Example from Myers:
	{"abcabba", "cbabac", []string{"caba", "baba", "cbba"}},
	{"3456aaa", "aaa", []string{"aaa"}},
	{"aaa", "aaa123", []string{"aaa"}},
	{"aabaa", "aacaa", []string{"aaaa"}},
	{"1a", "a", []string{"a"}},
	{"abab", "bb", []string{"bb"}},
	{"123", "ab", []string{""}},
	{"a", "b", []string{""}},
	{"abc", "123", []string{""}},
	{"aa", "aa", []string{"aa"}},
	{"abcde", "12345", []string{""}},
	{"aaa3456", "aaa", []string{"aaa"}},
	{"abcde", "12345a", []string{"a"}},
	{"ab", "123", []string{""}},
	{"1a2", "a", []string{"a"}},
	// for two-sided
	{"babaab", "cccaba", []string{"aba"}},
	{"aabbab", "cbcabc", []string{"bab"}},
	{"abaabb", "bcacab", []string{"baab"}},
	{"abaabb", "abaaaa", []string{"abaa"}},
	{"bababb", "baaabb", []string{"baabb"}},
	{"abbbaa", "cabacc", []string{"aba"}},
	{"aabbaa", "aacaba", []string{"aaaa", "aaba"}},
}

func init() {
	log.SetFlags(log.Lshortfile)
}

func lcslen(l lcs) int {
	ans := 0
	for _, d := range l {
		ans += int(d.Len)
	}
	return ans
}

// return a random string of length n made of characters from s
func randstr(s string, n int) string {
	src := []rune(s)
	x := make([]rune, n)
	for i := 0; i < n; i++ {
		x[i] = src[rand.Intn(len(src))]
	}
	return string(x)
}

func TestLcsFix(t *testing.T) {
	tests := []struct{ before, after lcs }{
		{lcs{diag{0, 0, 3}, diag{2, 2, 5}, diag{3, 4, 5}, diag{8, 9, 4}}, lcs{diag{0, 0, 2}, diag{2, 2, 1}, diag{3, 4, 5}, diag{8, 9, 4}}},
		{lcs{diag{1, 1, 6}, diag{6, 12, 3}}, lcs{diag{1, 1, 5}, diag{6, 12, 3}}},
		{lcs{diag{0, 0, 4}, diag{3, 5, 4}}, lcs{diag{0, 0, 3}, diag{3, 5, 4}}},
		{lcs{diag{0, 20, 1}, diag{0, 0, 3}, diag{1, 20, 4}}, lcs{diag{0, 0, 3}, diag{3, 22, 2}}},
		{lcs{diag{0, 0, 4}, diag{1, 1, 2}}, lcs{diag{0, 0, 4}}},
		{lcs{diag{0, 0, 4}}, lcs{diag{0, 0, 4}}},
		{lcs{}, lcs{}},
		{lcs{diag{0, 0, 4}, diag{1, 1, 6}, diag{3, 3, 2}}, lcs{diag{0, 0, 1}, diag{1, 1, 6}}},
	}
	for n, x := range tests {
		got := x.before.fix()
		if len(got) != len(x.after) {
			t.Errorf("got %v, expected %v, for %v", got, x.after, x.before)
		}
		olen := lcslen(x.after)
		glen := lcslen(got)
		if olen != glen {
			t.Errorf("%d: lens(%d,%d) differ, %v, %v, %v", n, glen, olen, got, x.after, x.before)
		}
	}
}

func TestRandOld(t *testing.T) {
	rand.Seed(1)
	for i := 0; i < 1000; i++ {
		// TODO(adonovan): use ASCII and bytesSeqs here? The use of
		// non-ASCII isn't relevant to the property exercised by the test.
		a := []rune(randstr("abω", 16))
		b := []rune(randstr("abωc", 16))
		seq := runesSeqs{a, b}

		const lim = 24 // large enough to get true lcs
		_, forw := compute(seq, forward, lim)
		_, back := compute(seq, backward, lim)
		_, two := compute(seq, twosided, lim)
		if lcslen(two) != lcslen(forw) || lcslen(forw) != lcslen(back) {
			t.Logf("\n%v\n%v\n%v", forw, back, two)
			t.Fatalf("%d forw:%d back:%d two:%d", i, lcslen(forw), lcslen(back), lcslen(two))
		}
		if !two.valid() || !forw.valid() || !back.valid() {
			t.Errorf("check failure")
		}
	}
}

// TestDiffAPI tests the public API functions (Diff{Bytes,Strings,Runes})
// to ensure at least minimal parity of the three representations.
func TestDiffAPI(t *testing.T) {
	for _, test := range []struct {
		a, b                              string
		wantStrings, wantBytes, wantRunes string
	}{
		{"abcXdef", "abcxdef", "[{3 4 3 4}]", "[{3 4 3 4}]", "[{3 4 3 4}]"}, // ASCII
		{"abcωdef", "abcΩdef", "[{3 5 3 5}]", "[{3 5 3 5}]", "[{3 4 3 4}]"}, // non-ASCII
	} {
		gotBytes := fmt.Sprint(DiffBytes([]byte(test.a), []byte(test.b)))
		if gotBytes != test.wantBytes {
			t.Errorf("DiffBytes(%q, %q) = %v, want %v",
				test.a, test.b, gotBytes, test.wantBytes)
		}
		gotRunes := fmt.Sprint(DiffRunes([]rune(test.a), []rune(test.b)))
		if gotRunes != test.wantRunes {
			t.Errorf("DiffRunes(%q, %q) = %v, want %v",
				test.a, test.b, gotRunes, test.wantRunes)
		}
	}
}

// This benchmark represents a common case for a diff command:
// large file with a single relatively small diff in the middle.
// (It's not clear whether this is representative of gopls workloads
// or whether it is important to gopls diff performance.)
//
// TODO(adonovan) opt: it could be much faster.  For example,
// comparing a file against itself is about 10x faster than with the
// small deletion in the middle. Strangely, comparing a file against
// itself minus the last byte is faster still; I don't know why.
// There is much low-hanging fruit here for further improvement.
func BenchmarkLargeFileSmallDiff(b *testing.B) {
	data, err := os.ReadFile("old.go") // large file
	if err != nil {
		log.Fatal(err)
	}

	n := len(data)

	src := string(data)
	dst := src[:n*49/100] + src[n*51/100:] // remove 2% from the middle

	srcBytes := []byte(src)
	dstBytes := []byte(dst)
	b.Run("bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			compute(bytesSeqs{srcBytes, dstBytes}, twosided, len(srcBytes)+len(dstBytes))
		}
	})

	srcRunes := []rune(src)
	dstRunes := []rune(dst)
	b.Run("runes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			compute(runesSeqs{srcRunes, dstRunes}, twosided, len(srcRunes)+len(dstRunes))
		}
	})
}
