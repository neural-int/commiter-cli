package main

import (
	"errors"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/contextinput"
	"github.com/natsuki0413/commiter-cli/internal/relation"
	"sort"
	"strings"
)

func canonicalFixture(f fixture) (fixture, map[string]string, error) {
	c := f
	c.Files = append([]contextinput.File(nil), f.Files...)
	sort.Slice(c.Files, func(i, j int) bool {
		if c.Files[i].NewPath == nil {
			return false
		}
		if c.Files[j].NewPath == nil {
			return true
		}
		a, b := *c.Files[i].NewPath, *c.Files[j].NewPath
		at, bt := strings.HasSuffix(a, "_test.go"), strings.HasSuffix(b, "_test.go")
		if at != bt {
			return !at
		}
		return a < b
	})
	toModel, restore := map[string]string{}, map[string]string{}
	paths := map[string]bool{}
	for i := range c.Files {
		file := &c.Files[i]
		if file.NewPath == nil || *file.NewPath == "" || paths[*file.NewPath] || file.ID == "" || toModel[file.ID] != "" {
			return c, nil, errors.New("invalid_canonical_input")
		}
		paths[*file.NewPath] = true
		model := fmt.Sprintf("S%03d", i+1)
		toModel[file.ID] = model
		restore[model] = file.ID
		file.ID = model
	}
	c.Graph.Edges = append([]relation.Relation(nil), f.Graph.Edges...)
	for i := range c.Graph.Edges {
		edge := &c.Graph.Edges[i]
		if toModel[edge.SourceID] == "" || toModel[edge.TargetID] == "" {
			return c, nil, errors.New("invalid_graph_binding")
		}
		edge.SourceID = toModel[edge.SourceID]
		edge.TargetID = toModel[edge.TargetID]
	}
	sort.Slice(c.Graph.Edges, func(i, j int) bool {
		a, b := c.Graph.Edges[i], c.Graph.Edges[j]
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.Kind < b.Kind
	})
	c.Expected = nil
	return c, restore, nil
}
