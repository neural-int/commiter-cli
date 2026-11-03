package historyattribution

import "context"

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
	s, e, ok, _ := containedLinesContext(context.Background(), contentLines(whole), contentLines(part))
	return s, e, ok
}

func containedLinesContext(ctx context.Context, a, b []EditLine) (int, int, bool, error) {
	if len(b) == 0 || len(b) > len(a) {
		return 0, 0, false, nil
	}
	for start := 0; start+len(b) <= len(a); start++ {
		if err := matchingCheckpoint(ctx); err != nil {
			return 0, 0, false, err
		}
		equal := true
		for j := range b {
			if j%128 == 0 {
				if err := matchingCheckpoint(ctx); err != nil {
					return 0, 0, false, err
				}
			}
			if a[start+j] != b[j] {
				equal = false
				break
			}
		}
		if equal {
			return start, start + len(b), true, nil
		}
	}
	return 0, 0, false, nil
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
	_ = markSplitMergeEvidenceContext(context.Background(), in, out)
}

func markSplitMergeEvidenceContext(ctx context.Context, in Input, out *Comparison) error {
	if err := matchingCheckpoint(ctx); err != nil {
		return err
	}
	type fragment struct {
		path      string
		block     Block
		result    int
		candidate Candidate
		lines     []EditLine
	}
	var current, history []fragment
	for _, f := range in.Files {
		if err := matchingCheckpoint(ctx); err != nil {
			return err
		}
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
					current = append(current, fragment{path: f.Path, block: b, result: i, lines: contentLines(b)})
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, op := range in.History.Operations {
		if err := matchingCheckpoint(ctx); err != nil {
			return err
		}
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
				history = append(history, fragment{path: op.Path, block: b, candidate: Candidate{op.Thread, op.Turn, op.Item, op.Index, false}, lines: contentLines(b)})
			}
		}
	}
	mark := func(c fragment, h fragment) error {
		r := &out.Results[c.result]
		if r.Status == "error" {
			return nil
		}
		addReason(r, "split_merge_mapping_uncertain")
		r.Status = "ambiguous"
		r.Reason = r.Reasons[0]
		found := false
		for _, candidate := range r.Candidates {
			if err := matchingCheckpoint(ctx); err != nil {
				return err
			}
			if candidate == h.candidate {
				found = true
			}
		}
		if !found {
			r.Candidates = append(r.Candidates, h.candidate)
		}
		turns := map[string]bool{}
		for _, candidate := range r.Candidates {
			if err := matchingCheckpoint(ctx); err != nil {
				return err
			}
			turns[candidate.Thread+"/"+candidate.Turn] = true
		}
		r.SingleTurn = len(turns) == 1
		return nil
	}
	type located struct {
		fragment   fragment
		start, end int
	}
	// Only whole fragments actually contained in the current counterpart can
	// participate in a pair. Cache their first range before the cross product.
	contained := func(whole fragment, parts []fragment) ([]located, error) {
		var found []located
		for _, part := range parts {
			if err := matchingCheckpoint(ctx); err != nil {
				return nil, err
			}
			if len(part.lines) == 0 || len(part.lines) > len(whole.lines) || part.lines[0].Direction != whole.lines[0].Direction {
				continue
			}
			s, e, ok, err := containedLinesContext(ctx, whole.lines, part.lines)
			if err != nil {
				return nil, err
			}
			if ok {
				found = append(found, located{part, s, e})
			}
		}
		return found, nil
	}
	// Split: disjoint fragments in two current paths occur in one history block.
	for _, h := range history {
		parts, err := contained(h, current)
		if err != nil {
			return err
		}
		for i, a := range parts {
			for _, b := range parts[i+1:] {
				if err := matchingCheckpoint(ctx); err != nil {
					return err
				}
				if a.fragment.path == b.fragment.path {
					continue
				}
				if a.end <= b.start || b.end <= a.start {
					if err := mark(a.fragment, h); err != nil {
						return err
					}
					if err := mark(b.fragment, h); err != nil {
						return err
					}
				}
			}
		}
	}
	// Merge: one current block contains disjoint whole history fragments from
	// distinct paths, at least one of which crosses the current file boundary.
	for _, c := range current {
		parts, err := contained(c, history)
		if err != nil {
			return err
		}
		for i, a := range parts {
			for _, b := range parts[i+1:] {
				if err := matchingCheckpoint(ctx); err != nil {
					return err
				}
				if a.fragment.path == b.fragment.path {
					continue
				}
				if a.end <= b.start || b.end <= a.start {
					if err := mark(c, a.fragment); err != nil {
						return err
					}
					if err := mark(c, b.fragment); err != nil {
						return err
					}
				}
			}
		}
	}
	return matchingCheckpoint(ctx)
}
