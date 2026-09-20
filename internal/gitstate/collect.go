package gitstate

import (
	"context"
	"sort"
)

func Collect(root string, options Options) (snapshot Snapshot, err error) {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if options.Files == nil {
		options.Files = osFiles{}
	}
	commonDir, err := gitPath(ctx, root, "--git-common-dir")
	if err != nil {
		return Snapshot{}, internal("cannot resolve common Git directory")
	}
	lock, err := acquireLock(commonDir)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() {
		if closeErr := lock.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	state, err := inspect(ctx, root)
	if err != nil {
		return Snapshot{}, err
	}

	rawChanges, err := readStatus(ctx, root, options.Pathspecs)
	if err != nil {
		return Snapshot{}, err
	}
	selected := make([]rawChange, 0, len(rawChanges))
	excluded := make([]Excluded, 0)
	candidateIndexes := make([]int, 0)
	candidates := make([]Candidate, 0)
	for _, change := range rawChanges {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		include, err := selectedByGlobs(change, options.Include, options.Exclude)
		if err != nil {
			return Snapshot{}, err
		}
		if !include {
			continue
		}
		level, reason, err := classifyChangePaths(change, options.AdditionalSensitiveGlobs)
		if err != nil {
			return Snapshot{}, err
		}
		if level == automaticallyExcluded {
			excluded = append(excluded, Excluded{Path: change.displayPath(), Reason: reason})
			continue
		}
		selected = append(selected, change)
		if level == sensitiveCandidate {
			candidateIndexes = append(candidateIndexes, len(selected)-1)
			candidates = append(candidates, Candidate{Path: change.displayPath(), Reason: reason})
		}
	}

	preapproved := make(map[ChangeIdentity]bool, len(options.ApprovedSensitiveChanges))
	for _, identity := range options.ApprovedSensitiveChanges {
		preapproved[identity] = true
	}
	approvedIndexes := make(map[int]bool, len(candidateIndexes))
	pendingIndexes := make([]int, 0, len(candidateIndexes))
	pendingCandidates := make([]Candidate, 0, len(candidates))
	for candidateOffset, selectedIndex := range candidateIndexes {
		change := selected[selectedIndex]
		if preapproved[ChangeIdentity{Status: change.status, OldPath: pointerValue(change.oldPath), NewPath: pointerValue(change.newPath)}] {
			approvedIndexes[selectedIndex] = true
			continue
		}
		pendingIndexes = append(pendingIndexes, selectedIndex)
		pendingCandidates = append(pendingCandidates, candidates[candidateOffset])
	}

	approved := false
	if len(pendingCandidates) > 0 && options.ApproveSensitiveCandidates != nil {
		approved, err = options.ApproveSensitiveCandidates(pendingCandidates)
		if err != nil {
			return Snapshot{}, internal("cannot obtain sensitive file approval")
		}
	}
	for candidateOffset, selectedIndex := range pendingIndexes {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		if approved {
			approvedIndexes[selectedIndex] = true
		} else {
			change := selected[selectedIndex]
			identity := ChangeIdentity{Status: change.status, OldPath: pointerValue(change.oldPath), NewPath: pointerValue(change.newPath)}
			excluded = append(excluded, Excluded{Path: pendingCandidates[candidateOffset].Path, Reason: SensitiveCandidateNotApprovedReason, Identity: &identity})
		}
	}

	filtered := make([]rawChange, 0, len(selected))
	filteredSensitive := make([]bool, 0, len(selected))
	for index, change := range selected {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		isCandidate := containsIndex(candidateIndexes, index)
		if isCandidate && !approvedIndexes[index] {
			continue
		}
		filtered = append(filtered, change)
		filteredSensitive = append(filteredSensitive, approvedIndexes[index])
	}
	sortRawWithSensitive(filtered, filteredSensitive)

	snapshot = Snapshot{
		Root:      root,
		Head:      state.head,
		Branch:    state.branch,
		Changes:   []Change{},
		Untracked: []string{},
		Excluded:  excluded,
	}
	if snapshot.IndexIdentity, err = indexIdentity(ctx, root); err != nil {
		return Snapshot{}, err
	}
	for index, change := range filtered {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		built, include, buildErr := buildChange(ctx, root, change, options.Files, filteredSensitive[index], state.objectFormat)
		if buildErr != nil {
			return Snapshot{}, buildErr
		}
		if !include {
			continue
		}
		built.ID = fileID(len(snapshot.Changes))
		snapshot.Changes = append(snapshot.Changes, built)
		if change.untracked {
			snapshot.Untracked = append(snapshot.Untracked, change.displayPath())
		}
	}
	sort.Strings(snapshot.Untracked)
	sort.Slice(snapshot.Excluded, func(i, j int) bool { return snapshot.Excluded[i].Path < snapshot.Excluded[j].Path })
	finalState, err := inspect(ctx, root)
	if err != nil {
		return Snapshot{}, err
	}
	finalIndex, err := indexIdentity(ctx, root)
	if err != nil {
		return Snapshot{}, err
	}
	if finalState.head != snapshot.Head || finalState.branch != snapshot.Branch || finalState.objectFormat != state.objectFormat || finalIndex != snapshot.IndexIdentity {
		return Snapshot{}, safety("Git state changed while collecting the snapshot")
	}
	return snapshot, nil
}

func classifyChangePaths(change rawChange, additional []string) (sensitivity, string, error) {
	paths := []*string{change.oldPath, change.newPath}
	result := notSensitive
	reason := ""
	for _, value := range paths {
		if value == nil {
			continue
		}
		level, pathReason, err := classifySensitive(*value, additional)
		if err != nil {
			return notSensitive, "", err
		}
		if level > result {
			result, reason = level, pathReason
		}
	}
	return result, reason, nil
}

func containsIndex(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortRawWithSensitive(changes []rawChange, sensitive []bool) {
	type pair struct {
		change    rawChange
		sensitive bool
	}
	pairs := make([]pair, len(changes))
	for index := range changes {
		pairs[index] = pair{change: changes[index], sensitive: sensitive[index]}
	}
	sort.Slice(pairs, func(i, j int) bool {
		left := pairs[i].change.displayPath() + "\x00" + pointerValue(pairs[i].change.oldPath) + "\x00" + pairs[i].change.status
		right := pairs[j].change.displayPath() + "\x00" + pointerValue(pairs[j].change.oldPath) + "\x00" + pairs[j].change.status
		return left < right
	})
	for index := range pairs {
		changes[index] = pairs[index].change
		sensitive[index] = pairs[index].sensitive
	}
}
