package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// MermaidOption configures Mermaid rendering.
type MermaidOption func(*mermaidConfig)

type mermaidConfig struct {
	direction               string // TD, LR, BT, RL
	showConditionIndicators bool   // add edge labels when conditions exist (generic "cond")
}

// WithDirection sets graph direction (e.g., "TD", "LR").
func WithDirection(dir string) MermaidOption {
	return func(c *mermaidConfig) {
		dir = strings.TrimSpace(strings.ToUpper(dir))
		switch dir {
		case "TD", "LR", "BT", "RL":
			c.direction = dir
		}
	}
}

// WithConditionIndicators toggles generic condition labels on edges when a condition exists.
func WithConditionIndicators(enabled bool) MermaidOption {
	return func(c *mermaidConfig) { c.showConditionIndicators = enabled }
}

// MermaidFlowchart renders the workflow graph as a Mermaid flowchart definition.
// The output starts with `graph TD` by default.
func (w *Workflow) MermaidFlowchart(opts ...MermaidOption) string {
	if w == nil || w.root == nil {
		return "graph TD\n"
	}

	cfg := mermaidConfig{direction: "TD"}
	for _, o := range opts {
		o(&cfg)
	}

	// Assign compact, stable ids per node pointer to avoid name collisions.
	type edge struct {
		from  string
		to    string
		label string // optional (e.g., condition)
	}
	nodes := make(map[string]string) // id -> display label
	edges := make([]edge, 0)

	stepIDs := make(map[*step]string)
	mergeIDs := make(map[*mergeStep]string)
	idSeq := 0
	nextID := func() string {
		idSeq++
		return fmt.Sprintf("n%d", idSeq)
	}

	ensureStep := func(s *step) string {
		if s == nil {
			return ""
		}
		if id, ok := stepIDs[s]; ok {
			return id
		}
		id := nextID()
		stepIDs[s] = id
		nodes[id] = s.name
		return id
	}
	ensureMerge := func(m *mergeStep) string {
		if m == nil {
			return ""
		}
		if id, ok := mergeIDs[m]; ok {
			return id
		}
		id := nextID()
		mergeIDs[m] = id
		nodes[id] = m.name
		return id
	}

	visitedSteps := make(map[*step]bool)
	visitedMerges := make(map[*mergeStep]bool)

	var findBranchTails func(s *step, accum map[*step]struct{})
	findBranchTails = func(s *step, accum map[*step]struct{}) {
		if s == nil {
			return
		}
		// Walk linear chain
		cur := s
		for cur != nil {
			if len(cur.branches) == 0 && cur.next == nil {
				accum[cur] = struct{}{}
				return
			}
			// If branches exist, tails are inside each child
			if len(cur.branches) > 0 {
				for _, child := range cur.branches {
					findBranchTails(child, accum)
				}
				// After exploring branches, if there's a merge and a following step,
				// tails are inside branches, not this node, so stop here.
				return
			}
			cur = cur.next
		}
	}

	var walk func(s *step)
	walk = func(s *step) {
		if s == nil {
			return
		}
		if visitedSteps[s] {
			return
		}
		visitedSteps[s] = true

		sid := ensureStep(s)

		// Linear edge to next
		if s.next != nil {
			tid := ensureStep(s.next)
			label := ""
			if cfg.showConditionIndicators && s.next.precond != nil {
				label = "cond"
			}
			edges = append(edges, edge{from: sid, to: tid, label: label})
		}

		// Branch edges
		if len(s.branches) > 0 {
			for i, child := range s.branches {
				tid := ensureStep(child)
				label := ""
				if cfg.showConditionIndicators && len(s.brConds) > i && s.brConds[i] != nil {
					label = "cond"
				}
				edges = append(edges, edge{from: sid, to: tid, label: label})
			}
			if s.merge != nil {
				mid := ensureMerge(s.merge)
				// Connect each branch tail to the merge node for clarity
				tails := make(map[*step]struct{})
				for _, child := range s.branches {
					findBranchTails(child, tails)
				}
				for t := range tails {
					tid := ensureStep(t)
					edges = append(edges, edge{from: tid, to: mid})
				}
				// Edge from merge to its next
				if s.merge.next != nil {
					nid := ensureStep(s.merge.next)
					edges = append(edges, edge{from: mid, to: nid})
				}
			}
		}

		// Recurse
		if s.next != nil {
			walk(s.next)
		}
		for _, child := range s.branches {
			walk(child)
		}
		if s.merge != nil && !visitedMerges[s.merge] {
			visitedMerges[s.merge] = true
			if s.merge.next != nil {
				walk(s.merge.next)
			}
		}
	}

	walk(w.root)

	// Stable output: sort nodes by id sequence order and edges by from/to
	nodeIDs := make([]string, 0, len(nodes))
	for id := range nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(i, j int) bool {
		// ids are n<number>
		return nodeIDs[i] < nodeIDs[j]
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from == edges[j].from {
			if edges[i].to == edges[j].to {
				return edges[i].label < edges[j].label
			}
			return edges[i].to < edges[j].to
		}
		return edges[i].from < edges[j].from
	})

	var b strings.Builder
	fmt.Fprintf(&b, "graph %s\n", cfg.direction)
	for _, id := range nodeIDs {
		// id["label"]
		label := nodes[id]
		// Escape quotes in label
		label = strings.ReplaceAll(label, "\"", "\\\"")
		fmt.Fprintf(&b, "%s[\"%s\"]\n", id, label)
	}
	for _, e := range edges {
		if e.label != "" {
			fmt.Fprintf(&b, "%s -->|%s| %s\n", e.from, e.label, e.to)
		} else {
			fmt.Fprintf(&b, "%s --> %s\n", e.from, e.to)
		}
	}
	return b.String()
}
