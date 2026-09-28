package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/contextinput"
)

func TestIssue141ClusterCandidateCoverage(t *testing.T) {
	for _, item := range append(issue140AtomicityFixtures(), issue141GuardrailFixtures()...) {
		prepared, ids, err := issue140Prepare(context.Background(), item, issue140ManyArms[5], 1024)
		if err != nil {
			t.Fatal(err)
		}
		blocks, err := issue141ClusterBlocks(prepared.Document, ids)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, block := range blocks {
			if len(block.IDs) == 0 || len(block.IDs) > 4 {
				t.Fatalf("%s: invalid block size: %+v", item.name, block)
			}
			for _, id := range block.IDs {
				if seen[id] {
					t.Fatalf("%s: duplicate file %s", item.name, id)
				}
				seen[id] = true
			}
		}
		if len(seen) != len(ids) {
			t.Fatalf("%s: incomplete blocks", item.name)
		}
		if item.name == "mixed_24" && len(blocks) != 6 {
			t.Fatalf("mixed_24: got %d blocks, want 6", len(blocks))
		}
		if item.name == "mixed_24" {
			gold := issue141GoldIndex(item.reference)
			partition := issue141Partition{}
			for _, group := range item.reference {
				partition.Groups = append(partition.Groups, issue141Group{GroupID: group[0], FileIDs: group})
			}
			clusters := issue141ClustersFromBlocks(blocks, partition)
			unitIDs := make([]string, len(clusters))
			for i := range clusters {
				clusters[i].ID = fmt.Sprintf("C%03d", i+1)
				unitIDs[i] = clusters[i].ID
			}
			pairs := issue141ClusterPairs(prepared.Document, clusters)
			decisions := make([]bool, len(pairs))
			for i, pair := range pairs {
				var a, b string
				for _, cluster := range clusters {
					if cluster.ID == pair.A {
						a = cluster.IDs[0]
					}
					if cluster.ID == pair.B {
						b = cluster.IDs[0]
					}
				}
				decisions[i] = gold[a] == gold[b]
			}
			unitGroups, _ := issue141BoundaryPartition(unitIDs, pairs, decisions)
			if !sameGroups(issue141ExpandClusterGroups(unitGroups, clusters), item.reference) {
				t.Fatal("mixed_24 gold groups are unreachable from cluster candidates")
			}
		}
		reversed := prepared.Document
		reversed.Files = append([]contextinput.File(nil), prepared.Document.Files...)
		for i, j := 0, len(reversed.Files)-1; i < j; i, j = i+1, j-1 {
			reversed.Files[i], reversed.Files[j] = reversed.Files[j], reversed.Files[i]
		}
		again, err := issue141ClusterBlocks(reversed, ids)
		if err != nil || !reflect.DeepEqual(blocks, again) {
			t.Fatalf("%s: blocks changed with Document file order", item.name)
		}
	}
}

func TestIssue141ClustersKeepLocalLabelsAndAllFiles(t *testing.T) {
	blocks := []issue141Block{{ID: "B001", IDs: []string{"F001", "F002"}}, {ID: "B002", IDs: []string{"F003", "F004"}}}
	partition := issue141Partition{Groups: []issue141Group{
		{GroupID: "A", FileIDs: []string{"F001", "F003"}},
		{GroupID: "B", FileIDs: []string{"F002", "F004"}},
	}}
	clusters := issue141ClustersFromBlocks(blocks, partition)
	if len(clusters) != 4 {
		t.Fatalf("labels from distinct blocks were merged: %+v", clusters)
	}
	for i := range clusters {
		clusters[i].ID = fmt.Sprintf("C%03d", i+1)
	}
	groups := issue141ExpandClusterGroups([][]string{{"C001", "C003"}, {"C002", "C004"}}, clusters)
	if !reflect.DeepEqual(groups, [][]string{{"F001", "F003"}, {"F002", "F004"}}) {
		t.Fatalf("wrong expansion: %+v", groups)
	}
}
