// Fixed-phase-input latency comparison. No Python proxy, download or Git access.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/natsuki0413/commiter-cli/internal/llm"
	"github.com/natsuki0413/commiter-cli/internal/mlx"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func save(path string, x any) {
	b, e := json.MarshalIndent(x, "", "  ")
	must(e)
	must(os.WriteFile(path, b, 0600))
}
func main() {
	helper := flag.String("helper", "", "fixed local helper")
	source := flag.String("source", "", "measured B phase inputs")
	out := flag.String("output", "", "new private directory")
	persistent := flag.Bool("persistent", false, "reuse one cycle")
	wire := flag.Bool("wire", false, "consume finalized phase messages without display conversion")
	independent := flag.Bool("independent-phases", false, "score each fixed phase independently; do not claim end-to-end propagation")
	phaseCount := flag.Int("phase-count", 3, "number of fixed phases, from Grouping onward (1-3)")
	flag.Parse()
	if *phaseCount < 1 || *phaseCount > 3 {
		panic("phase-count must be 1-3")
	}
	must(os.Mkdir(*out, 0700))
	client := mlx.NewClient(*helper)
	ctx := context.Background()
	generate := client.Generate
	if *persistent {
		s, e := client.BeginSession(ctx)
		must(e)
		defer s.Close()
		generate = s.Generate
	} else {
		client.Command = func(ctx context.Context, exe string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, exe, "--benchmark-single")
		}
	}
	rows := []any{}
	start := time.Now()
	for _, index := range []string{"01", "02", "03"}[:*phaseCount] {
		name := index + "-canonical-request.json"
		if *wire {
			name = index + "-request.json"
		}
		b, e := os.ReadFile(filepath.Join(*source, name))
		must(e)
		var request mlx.Request
		must(json.Unmarshal(b, &request))
		if !*wire {
			content, e := llm.LiteralNewlineDisplay([]byte(request.Messages[1].Content))
			must(e)
			request.Messages[1].Content = content
		}
		save(filepath.Join(*out, index+"-request.json"), request)
		t := time.Now()
		response, e := generate(ctx, request)
		save(filepath.Join(*out, index+"-response.json"), response)
		must(e)
		rows = append(rows, map[string]any{"phase": index, "wall_seconds": time.Since(t).Seconds(), "metrics": response.BenchmarkMetrics})
		save(filepath.Join(*out, "results.json"), rows)
		// Later fixed inputs are valid only while upstream outputs equal their source.
		if index != "03" && !*independent {
			previous, e := os.ReadFile(filepath.Join(*source, index+"-response.json"))
			must(e)
			var old mlx.Response
			must(json.Unmarshal(previous, &old))
			var a, b any
			must(json.Unmarshal([]byte(old.GeneratedJSON), &a))
			must(json.Unmarshal([]byte(response.GeneratedJSON), &b))
			if !reflect.DeepEqual(a, b) {
				save(filepath.Join(*out, "completion.json"), map[string]any{"state": "stopped_upstream_changed", "phase": index})
				fmt.Println("stopped: upstream differs from fixed downstream input")
				return
			}
		}
	}
	save(filepath.Join(*out, "completion.json"), map[string]any{"state": "complete", "cells": *phaseCount, "wall_seconds": time.Since(start).Seconds(), "persistent": *persistent})
	fmt.Println("complete")
}
