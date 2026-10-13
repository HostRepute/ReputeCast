package demo

import (
	"math"
	"sort"
)

// Helpers for the reports seeder: integer tables whose rows and columns add
// up to given totals, so every breakdown of a report (mailbox provider,
// sending provider, A/B variant) agrees with the headline counters.

// reportMargins turns rates (percent of base) into integer targets that add
// up to total. Each target first stays inside the interval that still shows
// the board's rounded rate (digits decimals); what is left goes to the
// biggest slice when it is small, else to every slice by size.
func reportMargins(rates []float64, bases []int64, total int64, digits int) []int64 {
	n := len(rates)
	out := make([]int64, n)
	half := 0.5 * math.Pow(10, -float64(digits))
	lo := make([]int64, n)
	hi := make([]int64, n)
	var sum int64
	for i := range rates {
		b := float64(bases[i])
		out[i] = int64(math.Round(rates[i] / 100 * b))
		lo[i] = int64(math.Ceil((rates[i] - half) / 100 * b))
		hi[i] = int64(math.Floor((rates[i] + half) / 100 * b))
		if hi[i] > lo[i] && float64(hi[i])/b*100 >= rates[i]+half-1e-12 {
			hi[i]-- // the upper bound itself would round up
		}
		lo[i] = max(lo[i], 0)
		if out[i] < lo[i] {
			out[i] = lo[i]
		}
		if out[i] > hi[i] && hi[i] >= lo[i] {
			out[i] = hi[i]
		}
		sum += out[i]
	}
	// Order by size: residual units go to the biggest bases first.
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return bases[order[a]] > bases[order[b]] })
	for r := total - sum; r != 0; {
		moved := false
		for _, i := range order {
			if r > 0 && out[i] < hi[i] {
				out[i]++
				r--
				moved = true
			} else if r < 0 && out[i] > lo[i] {
				out[i]--
				r++
				moved = true
			}
			if r == 0 {
				break
			}
		}
		if moved {
			continue
		}
		// No room left inside the intervals: one slice takes the rest (the
		// biggest), so every other slice still shows the board's rate.
		out[order[0]] += r
		break
	}
	return out
}

// reportFit2D fills a table whose row sums are rows and column sums are
// cols, starting from init (iterative proportional fitting), then rounds it
// so both sums hold exactly.
func reportFit2D(rows, cols []int64, init [][]float64) [][]int64 {
	nr, nc := len(rows), len(cols)
	x := make([][]float64, nr)
	for i := range x {
		x[i] = make([]float64, nc)
		for j := range x[i] {
			x[i][j] = math.Max(init[i][j], 0)
			if x[i][j] == 0 && rows[i] > 0 && cols[j] > 0 {
				x[i][j] = 1e-9
			}
		}
	}
	for it := 0; it < 200; it++ {
		for i := range x {
			var s float64
			for j := range x[i] {
				s += x[i][j]
			}
			if s > 0 {
				f := float64(rows[i]) / s
				for j := range x[i] {
					x[i][j] *= f
				}
			}
		}
		worst := 0.0
		for j := 0; j < nc; j++ {
			var s float64
			for i := range x {
				s += x[i][j]
			}
			if s > 0 {
				f := float64(cols[j]) / s
				worst = math.Max(worst, math.Abs(f-1))
				for i := range x {
					x[i][j] *= f
				}
			}
		}
		if worst < 1e-12 {
			break
		}
	}
	return reportRound2D(x, rows, cols)
}

// reportRound2D rounds a real table to integers keeping the given row and
// column sums: everything is floored, then the missing units are placed
// along augmenting paths (a small max-flow), largest fractions first.
func reportRound2D(x [][]float64, rows, cols []int64) [][]int64 {
	nr, nc := len(rows), len(cols)
	out := make([][]int64, nr)
	rowNeed := make([]int64, nr)
	colNeed := make([]int64, nc)
	copy(colNeed, cols)
	type cell struct {
		i, j int
		f    float64
	}
	var cells []cell
	for i := range x {
		out[i] = make([]int64, nc)
		rowNeed[i] = rows[i]
		for j := range x[i] {
			v := math.Floor(x[i][j] + 1e-9)
			out[i][j] = int64(v)
			rowNeed[i] -= out[i][j]
			colNeed[j] -= out[i][j]
			cells = append(cells, cell{i, j, x[i][j] - v})
		}
	}
	sort.SliceStable(cells, func(a, b int) bool { return cells[a].f > cells[b].f })
	// extra[i][j] is the unit added on top of the floor (0 or 1, or more
	// when the fit was off by more than one).
	extra := make([][]int64, nr)
	for i := range extra {
		extra[i] = make([]int64, nc)
	}
	// Greedy first.
	for _, c := range cells {
		if rowNeed[c.i] > 0 && colNeed[c.j] > 0 && extra[c.i][c.j] == 0 {
			extra[c.i][c.j]++
			rowNeed[c.i]--
			colNeed[c.j]--
		}
	}
	// Then augmenting paths: row i with need → column j (add) → row k that
	// already took a unit in j (move it) → … → a column with need.
	for i := 0; i < nr; i++ {
		for rowNeed[i] > 0 {
			seenRow := make([]bool, nr)
			var dfs func(r int) bool
			dfs = func(r int) bool {
				seenRow[r] = true
				for j := 0; j < nc; j++ {
					if colNeed[j] > 0 {
						extra[r][j]++
						colNeed[j]--
						return true
					}
				}
				for j := 0; j < nc; j++ {
					for k := 0; k < nr; k++ {
						if !seenRow[k] && extra[k][j] > 0 {
							// Move k's unit in column j to r, then find k another column.
							extra[r][j]++
							extra[k][j]--
							if dfs(k) {
								return true
							}
							extra[r][j]--
							extra[k][j]++
						}
					}
				}
				return false
			}
			if !dfs(i) {
				// Totals don't match (shouldn't happen): give up on this row.
				break
			}
			rowNeed[i]--
		}
	}
	for i := range out {
		for j := range out[i] {
			out[i][j] += extra[i][j]
		}
	}
	return out
}
