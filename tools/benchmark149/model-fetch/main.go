// model-fetch is an explicitly approved, pinned benchmark capability download.
// It sends only public model identifiers to the registry, never repository data.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/natsuki0413/commiter-cli/internal/mlxmodel"
)

func main() {
	approved := flag.Bool("approved", false, "human approved this exact model acquisition")
	flag.Parse()
	if !*approved {
		fmt.Fprintln(os.Stderr, "explicit model acquisition approval required")
		os.Exit(1)
	}
	store, err := mlxmodel.DefaultStore()
	if err != nil {
		panic(err)
	}
	spec := mlxmodel.Spec{Repo: "mlx-community/Qwen3-8B-4bit", Revision: "545dc4251c05440727734bcd94334791f6ab0192", Quantization: "4bit"}
	// Acquisition can take longer than an inference run on a slow connection.
	// Preserve Store's HTTPS registry redirect boundary with a longer timeout.
	store.Client = &http.Client{Timeout: 2 * time.Hour, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		host := strings.ToLower(req.URL.Hostname())
		if len(via) >= 5 || req.URL.Scheme != "https" ||
			(host != "huggingface.co" && !strings.HasSuffix(host, ".huggingface.co") &&
				host != "hf.co" && !strings.HasSuffix(host, ".hf.co")) {
			return errors.New("model download redirected outside the registry")
		}
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	start := time.Now()
	plan, err := store.Plan(ctx, spec)
	if err != nil {
		panic(err)
	}
	if plan.Bytes != 4623782544 || len(plan.Files) != 9 {
		panic("pinned download differs from approved size/file inventory")
	}
	fmt.Fprintf(os.Stderr, "approved pinned acquisition: %d bytes, %d files\n", plan.Bytes, len(plan.Files))
	if _, err = store.Install(ctx, plan); err != nil {
		panic(err)
	}
	if _, err = store.Ready(spec); err != nil {
		panic(err)
	}
	result := struct {
		Model                   mlxmodel.Spec `json:"model"`
		Bytes                   int64         `json:"bytes"`
		Files                   int           `json:"files"`
		Ready                   bool          `json:"ready"`
		DigestVerifiedOnInstall bool          `json:"digest_verified_on_install"`
		Wall                    float64       `json:"wall_seconds"`
	}{spec, plan.Bytes, len(plan.Files), true, true, time.Since(start).Seconds()}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
