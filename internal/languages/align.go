package languages

// Align matches the blocks of an earlier version of a document to a later
// one, so a block keeps its identity while it is edited or moved. Blocks are
// matched in three passes:
//
//  1. identical blocks, in order (the longest common subsequence of the
//     exact fingerprints), then identical blocks that moved;
//  2. between two matched neighbours, blocks whose translatable text is the
//     same (a style or link change);
//  3. what is left between the same neighbours, in order: a block edited in
//     place.
//
// It returns, for each later block, the index of the earlier block it
// continues, or -1 for a new block. Earlier blocks no later block continues
// were removed.
func Align(before, after []AlignKey) []int {
	out := make([]int, len(after))
	for i := range out {
		out[i] = -1
	}
	usedBefore := make([]bool, len(before))
	// ordered marks later blocks matched in order: they bound the gaps.
	ordered := make([]bool, len(after))
	// 1. Exact matches in order.
	for _, p := range lcs(before, after, func(a, b AlignKey) bool { return a.Raw != "" && a.Raw == b.Raw }) {
		out[p[1]], usedBefore[p[0]], ordered[p[1]] = p[0], true, true
	}
	// Exact matches that moved.
	for j := range after {
		if out[j] >= 0 || after[j].Raw == "" {
			continue
		}
		for i := range before {
			if !usedBefore[i] && before[i].Raw == after[j].Raw {
				out[j], usedBefore[i] = i, true
				break
			}
		}
	}
	// 2 and 3, gap by gap between blocks matched in order.
	gapPass := func(match func(a, b AlignKey) bool) {
		lastBefore := -1
		j := 0
		for j < len(after) {
			if ordered[j] {
				lastBefore = out[j]
				j++
				continue
			}
			end := j
			for end < len(after) && !ordered[end] {
				end++
			}
			nextBefore := len(before)
			if end < len(after) {
				nextBefore = out[end]
			}
			k := lastBefore + 1
			for g := j; g < end; g++ {
				if out[g] >= 0 {
					continue // moved
				}
				for i := k; i < nextBefore && i < len(before); i++ {
					if !usedBefore[i] && match(before[i], after[g]) {
						out[g], usedBefore[i], ordered[g] = i, true, true
						k = i + 1
						break
					}
				}
			}
			j = end
		}
	}
	gapPass(func(a, b AlignKey) bool { return a.Hash != "" && a.Hash == b.Hash })
	// Blocks that moved and changed style keep their text.
	for j := range after {
		if out[j] >= 0 || after[j].Hash == "" {
			continue
		}
		for i := range before {
			if !usedBefore[i] && before[i].Hash == after[j].Hash {
				out[j], usedBefore[i] = i, true
				break
			}
		}
	}
	gapPass(func(a, b AlignKey) bool { return true })
	return out
}

// AlignKey is what Align compares: a block's exact and textual fingerprints.
type AlignKey struct {
	Raw  string
	Hash string
}

// lcs returns index pairs of a longest common subsequence.
func lcs(a, b []AlignKey, eq func(x, y AlignKey) bool) [][2]int {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if eq(a[i], b[j]) {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	var out [][2]int
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case eq(a[i], b[j]):
			out = append(out, [2]int{i, j})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			j++
		}
	}
	return out
}

// keysOf returns the align keys of blocks.
func keysOf(blocks []Block) []AlignKey {
	out := make([]AlignKey, len(blocks))
	for i, b := range blocks {
		out[i] = AlignKey{Raw: b.Raw, Hash: b.Hash}
	}
	return out
}
