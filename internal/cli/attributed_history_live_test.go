package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/config"
	"github.com/natsuki0413/commiter-cli/internal/gitstate"
	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

// Opt-in, read-only integration: uses an explicitly supplied existing fixture
// and installed helper/model. It never exports history, downloads or writes Git.
func TestInstalledModelWithLiveAttributedHistory(t *testing.T) {
	root, helper := os.Getenv("COMMITER_LIVE_ATTRIBUTED_FIXTURE"), os.Getenv("COMMITER_LIVE_ATTRIBUTED_HELPER")
	if root == "" || helper == "" {
		t.Skip("explicit existing fixture and installed helper required")
	}
	paths := []string{"src/components/common/navigation/NavigationHrefLink.jsx", "src/components/common/sitefooter/SiteFooter.jsx", "src/components/common/sitefooter/SiteFooter.test.jsx"}
	snapshot, err := gitstate.Collect(root, gitstate.Options{Pathspecs: paths})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Changes) != 3 {
		t.Fatal("expected the three explicit fixture changes")
	}
	oldHelper := mlxHelperPath
	t.Cleanup(func() { mlxHelperPath = oldHelper })
	mlxHelperPath = func() (string, error) { return helper, nil }
	// This is the parent test budget, not a history-specific collection deadline.
	parent, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cycle := &codexCycle{started: time.Now()}
	ctx := context.WithValue(parent, codexCycleKey{}, cycle)
	started := time.Now()
	plan, err := generateCommitPlan(ctx, root, snapshot, config.Defaults().Values, "")
	if err != nil {
		t.Fatal(err)
	}
	if !cycle.attributionCollected || cycle.attribution.Contexts == 0 || cycle.attribution.History == nil {
		t.Fatal("no live attributed final-response history reached the normal planning path")
	}
	states := map[string]int{}
	for _, file := range cycle.attribution.History.Files {
		for _, block := range file.Blocks {
			states[block.Status]++
		}
	}
	if states["matched"]+states["ambiguous"] == 0 {
		t.Fatal("live history contains no content-attributed candidates")
	}
	if len(plan.Commits) == 0 {
		t.Fatal("final validator returned no plan")
	}
	if err := verifyAttributionSnapshot(parent, root, snapshot, config.Defaults().Values); err != nil {
		t.Fatal(err)
	}
	t.Logf("live history contexts=%d operations=%d blocks matched=%d ambiguous=%d unmatched=%d error=%d commits=%d elapsed=%s; Git unchanged", cycle.attribution.Contexts, cycle.attribution.Operations, states["matched"], states["ambiguous"], states["unmatched"], states["error"], len(plan.Commits), time.Since(started))
}

func TestTooManyFilesStopsBeforeHistoryOrModelPreparation(t *testing.T) {
	oldStore := newMLXModelStore
	t.Cleanup(func() { newMLXModelStore = oldStore })
	newMLXModelStore = func() (mlxmodel.Store, error) {
		t.Fatal("oversized input started model preparation")
		return mlxmodel.Store{}, nil
	}
	snapshot := gitstate.Snapshot{Changes: make([]gitstate.Change, 5)}
	if _, err := generateCommitPlan(context.Background(), "", snapshot, config.Defaults().Values, ""); err == nil {
		t.Fatal("five-file normal input accepted")
	}
}
