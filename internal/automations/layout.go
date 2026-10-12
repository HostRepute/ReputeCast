package automations

import "math"

// Layout places steps the way the flow board does: the trigger at the top of
// a 900px frame, a branch's paths 220px to each side (120px for a split
// inside a path), steps stacked with 18px between them and a merge back under
// its parents. It is used for templates and seeded flows; on the canvas people
// move steps freely.

const (
	layoutSpineX   = 370.0
	layoutTop      = 24.0
	layoutGap      = 18.0
	layoutForkGap  = 54.0
	layoutMergeGap = 38.0
	nodeWidth      = 224.0
)

// estimatedHeight is a generous height for a step card, so laid-out steps
// never overlap once the canvas measures them.
func estimatedHeight(n FlowNode) float64 {
	switch n.Type {
	case NodeTrigger:
		return 104
	case NodeDelay, NodeSplit:
		return 56
	case NodeGoal, NodeExit:
		return 76
	case NodeEmail:
		return 82
	}
	return 80
}

// Layout returns g with every node positioned.
func Layout(g FlowGraph) FlowGraph {
	order := g.topo()
	x := map[string]float64{}
	y := map[string]float64{}
	spread := map[string]float64{}
	if t := g.Trigger(); t != nil {
		x[t.ID] = layoutSpineX
		y[t.ID] = layoutTop
		spread[t.ID] = 220
	}
	for _, id := range order {
		n := g.Node(id)
		if _, ok := x[id]; !ok {
			// Unreachable steps sit on the spine.
			x[id] = layoutSpineX
			y[id] = layoutTop
			spread[id] = 220
		}
		out := g.Out(id)
		fork := n.Type == NodeBranch || n.Type == NodeSplit
		for i, e := range out {
			cx := x[id]
			childSpread := spread[id]
			gap := layoutGap
			if fork && len(out) > 1 {
				offset := (float64(i) - float64(len(out)-1)/2) * 2 * spread[id]
				cx = x[id] + offset
				childSpread = 120
				gap = layoutForkGap
				if spread[id] < 220 {
					gap = 44
				}
			}
			if len(g.In(e.Target)) > 1 {
				gap = math.Max(gap, layoutMergeGap)
			}
			cy := y[id] + estimatedHeight(*n) + gap
			if prev, ok := y[e.Target]; ok {
				// A merge: centre under all parents, below the lowest.
				ins := g.In(e.Target)
				sum := 0.0
				for _, in := range ins {
					sum += x[in.Source]
				}
				x[e.Target] = sum / float64(len(ins))
				y[e.Target] = math.Max(prev, cy)
				continue
			}
			x[e.Target] = cx
			y[e.Target] = cy
			spread[e.Target] = childSpread
		}
	}
	out := FlowGraph{Nodes: make([]FlowNode, len(g.Nodes)), Edges: g.Edges}
	for i, n := range g.Nodes {
		n.Position = FlowPosition{X: math.Round(x[n.ID]), Y: math.Round(y[n.ID])}
		out.Nodes[i] = n
	}
	return out
}
