package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/gitstate"
)

func TestPreregisteredBoundaryUsesSameBaseBytes(t *testing.T) {
	all := fixtures()
	if len(all) != 8 || !reflect.DeepEqual(all[3].Files, all[4].Files[:4]) || len(all[4].Files) != 5 || all[4].Files[4].Intent != all[3].Files[0].Intent {
		t.Fatal("4→5 workload no longer represents the same base plus one dependent test")
	}
}

func TestBenchmarkPreservesRealSnapshotThroughMetadataAndReplay(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("numeric helper measurement requires macOS time -l")
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	f := fixtures()[1]
	root, err := makeRepository(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	helper := filepath.Join(t.TempDir(), "measurement-fixture")
	// A protocol double verifies the complete host path. Its numeric constants
	// are never used as model measurements or written into evaluation artifacts.
	const protocol = `#!/usr/bin/python3
import sys,json
r=json.load(sys.stdin)
p=json.loads(r['messages'][1]['content'])
answers={}
for g in p['groups']:
 if r['generation_profile']=='bounded-category':
  kind='test' if g['files'][0]['new_path'].endswith('_test.go') else 'fix'
  answers[g['id']]={'type':kind,'breaking_evidence_ref':'none'}
 else:
  answers[g['id']]={'scope':'settings','summary':'include the boundary value'}
print(json.dumps({'ok':True,'stop_reason':'completed','generated_json':json.dumps(answers),'model':r['model'],'runtime':'mlx','error_class':None,'generation_profile':r['generation_profile'],'benchmark_input_tokens':0,'benchmark_output_tokens':0,'benchmark_load_seconds':0,'benchmark_ttft_seconds':0,'benchmark_peak_bytes':0}))
`
	if err = os.WriteFile(helper, []byte(protocol), 0700); err != nil {
		t.Fatal(err)
	}
	snapshot, err := gitstate.Collect(root, gitstate.Options{})
	if err != nil {
		t.Fatal(err)
	}
	p, bytes, lines, err := prepare(root, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	m := run(f, root, snapshot, p, "file-first", 0, bytes, lines, &measuredBackend{Helper: helper, Model: "protocol-double"})
	if m.Status != "completed" || !m.GitUnchanged || !m.BoundaryUnchanged || m.Groups != 2 || len(m.Calls) != 2 || m.Quality == nil || m.Quality.FS != 1 || m.Quality.Exact {
		t.Fatalf("measurement contract=%#v", m)
	}
}
