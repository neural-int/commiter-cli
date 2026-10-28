package historyattribution

// Cross-file evidence is limited to fully contained, contiguous add/delete
// fragments in the explicitly selected paths. This records uncertainty, not an
// inferred move, replayed snapshot, or unique causal assignment.
func contentLines(b Block) []EditLine {
	var out []EditLine
	for _, line := range b.Lines {
		if line.Direction != "eof" {
			out = append(out, line)
		}
	}
	return out
}
func containedRange(whole, part Block) (int, int, bool) {
	a, b := contentLines(whole), contentLines(part)
	if len(b) == 0 || len(b) > len(a) {
		return 0, 0, false
	}
	for start := 0; start+len(b) <= len(a); start++ {
		equal := true
		for j := range b {
			if a[start+j] != b[j] {
				equal = false
				break
			}
		}
		if equal {
			return start, start + len(b), true
		}
	}
	return 0, 0, false
}
func singleDirection(b Block) bool {
	lines := contentLines(b)
	if len(lines) == 0 {
		return false
	}
	for _, line := range lines {
		if line.Direction != lines[0].Direction {
			return false
		}
	}
	return true
}
func markSplitMergeEvidence(in Input, out *Comparison) {
	type fragment struct {
		path      string
		block     Block
		result    int
		candidate Candidate
	}
	var current, history []fragment
	for _, f := range in.Files {
		kind := "update"
		if f.Kind == "untracked" {
			kind = "add"
		}
		if f.Kind == "rename" {
			continue
		}
		bs, e := comparisonBlocks(kind, f.Diff, "")
		if e != nil {
			continue
		}
		for n, b := range bs {
			if !singleDirection(b) {
				continue
			}
			for i, r := range out.Results {
				if r.Path == f.Path && r.Block == n {
					current = append(current, fragment{path: f.Path, block: b, result: i})
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, op := range in.History.Operations {
		if !op.Completed || op.Diff == nil || op.Kind == "rename" || seen[operationKey(op)] {
			continue
		}
		seen[operationKey(op)] = true
		bs, e := comparisonBlocks(op.Kind, *op.Diff, "")
		if e != nil {
			continue
		}
		for _, b := range bs {
			if singleDirection(b) {
				history = append(history, fragment{path: op.Path, block: b, candidate: Candidate{op.Thread, op.Turn, op.Item, op.Index, false}})
			}
		}
	}
	mark := func(c fragment, h fragment) {
		r := &out.Results[c.result]
		if r.Status == "error" {
			return
		}
		addReason(r, "split_merge_mapping_uncertain")
		r.Status = "ambiguous"
		r.Reason = r.Reasons[0]
		found := false
		for _, candidate := range r.Candidates {
			if candidate == h.candidate {
				found = true
			}
		}
		if !found {
			r.Candidates = append(r.Candidates, h.candidate)
		}
		turns := map[string]bool{}
		for _, candidate := range r.Candidates {
			turns[candidate.Thread+"/"+candidate.Turn] = true
		}
		r.SingleTurn = len(turns) == 1
	}
	// Split: disjoint fragments in two current paths occur in one history block.
	for _, h := range history {
		for i, a := range current {
			for _, b := range current[i+1:] {
				if a.path == b.path || (a.path == h.path && b.path == h.path) {
					continue
				}
				as, ae, aok := containedRange(h.block, a.block)
				bs, be, bok := containedRange(h.block, b.block)
				if aok && bok && (ae <= bs || be <= as) {
					mark(a, h)
					mark(b, h)
				}
			}
		}
	}
	// Merge: one current block contains disjoint whole history fragments from
	// distinct paths, at least one of which crosses the current file boundary.
	for _, c := range current {
		for i, a := range history {
			for _, b := range history[i+1:] {
				if a.path == b.path || (a.path == c.path && b.path == c.path) {
					continue
				}
				as, ae, aok := containedRange(c.block, a.block)
				bs, be, bok := containedRange(c.block, b.block)
				if aok && bok && (ae <= bs || be <= as) {
					mark(c, a)
					mark(c, b)
				}
			}
		}
	}
}
