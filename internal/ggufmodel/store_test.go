package ggufmodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testModel = "GGUF public synthetic model bytes"

func testSpec() Spec {
	h := sha256.Sum256([]byte(testModel))
	return Spec{Repo: "owner/model", Revision: strings.Repeat("a", 40), Quantization: "Q4_K_M", Files: []File{{Path: "model.gguf", Size: int64(len(testModel)), SHA256: hex.EncodeToString(h[:])}}}
}

func fetchTestModel(context.Context, Spec, File) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(testModel)), nil
}

func testStore(t *testing.T) Store {
	t.Helper()
	return Store{Root: filepath.Join(t.TempDir(), "store")}
}
func requireReady(t *testing.T, s Store) {
	t.Helper()
	lease, err := s.Open(context.Background(), testSpec())
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
func install(t *testing.T, s Store) {
	t.Helper()
	if err := s.Install(context.Background(), testSpec(), false, fetchTestModel); err != nil {
		t.Fatal(err)
	}
}
func corrupt(t *testing.T, s Store) {
	t.Helper()
	key, _ := CacheKey(testSpec())
	if err := os.WriteFile(filepath.Join(s.Root, key, "model.gguf"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCacheIdentityGoldenAndRegistrationValidation(t *testing.T) {
	spec := testSpec()
	key, err := CacheKey(spec)
	if err != nil || key != "528a0ae7c6ec28c6ababd6e8e820d597f0bbf09a5cd09422936eaec88717a464" {
		t.Fatalf("key=%s err=%v", key, err)
	}
	upper := spec
	upper.Revision = strings.ToUpper(spec.Revision)
	other, _ := CacheKey(upper)
	if other != key {
		t.Fatal("revision not canonicalized")
	}
	for _, change := range []func(*Spec){func(s *Spec) { s.Repo = "owner/other" }, func(s *Spec) { s.Revision = strings.Repeat("b", 40) }, func(s *Spec) { s.Quantization = "Q8_0" }, func(s *Spec) { s.Files = append([]File(nil), s.Files...); s.Files[0].Path = "renamed.gguf" }} {
		changed := spec
		change(&changed)
		other, err := CacheKey(changed)
		if err != nil || other == key {
			t.Fatalf("identity unchanged: %v", err)
		}
	}
	for _, path := range []string{"", "../model.gguf", "/model.gguf", "sub/../model.gguf", "sub//model.gguf", ".locks/model.gguf", "commiter-model.json", "model\x00.gguf", "a\\model.gguf"} {
		invalid := testSpec()
		invalid.Files[0].Path = path
		if err := Validate(invalid); err == nil {
			t.Errorf("accepted invalid artifact name")
		}
	}
	for _, change := range []func(*Spec){func(s *Spec) { s.Files[0].SHA256 = "" }, func(s *Spec) { s.Files[0].Size = 0 }, func(s *Spec) { s.Revision = "main" }, func(s *Spec) { s.Quantization = "Q4\x00_K_M" }, func(s *Spec) { s.Files = append(s.Files, s.Files[0]) }} {
		invalid := testSpec()
		change(&invalid)
		if err := Validate(invalid); err == nil {
			t.Fatal("accepted invalid registration")
		}
	}
}

func TestInstallVerifiesAndReusesPinnedCache(t *testing.T) {
	s := testStore(t)
	calls := 0
	fetch := func(ctx context.Context, spec Spec, f File) (io.ReadCloser, error) {
		calls++
		return fetchTestModel(ctx, spec, f)
	}
	for _, update := range []bool{false, false, true} {
		if err := s.Install(context.Background(), testSpec(), update, fetch); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("healthy cache fetched %d times", calls)
	}
	requireReady(t, s)
	corrupt(t, s)
	if _, err := s.Open(context.Background(), testSpec()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("corrupt read accepted: %v", err)
	}
	if err := s.Install(context.Background(), testSpec(), false, fetch); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("implicit repair: %v", err)
	}
	if err := s.Install(context.Background(), testSpec(), true, fetch); err != nil {
		t.Fatal(err)
	}
	requireReady(t, s)
}

func TestDigestFailureCannotCommitOrDestroyPreviousCache(t *testing.T) {
	for _, repair := range []bool{false, true} {
		s := testStore(t)
		if repair {
			install(t, s)
			corrupt(t, s)
		}
		bad := func(context.Context, Spec, File) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(strings.Repeat("x", len(testModel)))), nil
		}
		if err := s.Install(context.Background(), testSpec(), repair, bad); !errors.Is(err, ErrCorrupt) {
			t.Fatal(err)
		}
		key, _ := CacheKey(testSpec())
		if _, err := os.Lstat(filepath.Join(s.Root, ".stage-"+key)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("failed stage retained")
		}
		if repair {
			data, err := os.ReadFile(filepath.Join(s.Root, key, "model.gguf"))
			if err != nil || string(data) != "corrupt" {
				t.Fatalf("previous content lost: %v", err)
			}
		} else if _, err := s.Open(context.Background(), testSpec()); !errors.Is(err, ErrMissing) {
			t.Fatal(err)
		}
	}
}

func TestCancelledDownloadPreservesCauseAndDoesNotCommit(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	err := s.Install(ctx, testSpec(), false, func(context.Context, Spec, File) (io.ReadCloser, error) {
		cancel()
		return nil, context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("download cancellation lost: %v", err)
	}
	if _, err := s.Open(context.Background(), testSpec()); !errors.Is(err, ErrMissing) {
		t.Fatalf("cancelled model committed: %v", err)
	}
}

func TestLeaseBlocksRepairAndCancellationDoesNotMutate(t *testing.T) {
	s := testStore(t)
	install(t, s)
	lease, err := s.Open(context.Background(), testSpec())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	key, _ := CacheKey(testSpec())
	lockPath := filepath.Join(s.Root, ".locks", key+".lock")
	before, _ := os.Stat(lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	err = s.Install(ctx, testSpec(), true, func(context.Context, Spec, File) (io.ReadCloser, error) {
		t.Fatal("repair acquired active reader lock")
		return nil, nil
	})
	if !errors.Is(err, ErrBusy) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait classification: %v", err)
	}
	after, _ := os.Stat(lockPath)
	if !os.SameFile(before, after) {
		t.Fatal("lock inode changed")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Install(context.Background(), testSpec(), true, fetchTestModel); err != nil {
		t.Fatal(err)
	}
	requireReady(t, s)
}

func TestCacheRefusesSymlinksAndUnexpectedRecoveryState(t *testing.T) {
	s := testStore(t)
	install(t, s)
	key, _ := CacheKey(testSpec())
	out := t.TempDir()
	if err := os.Symlink(out, filepath.Join(s.Root, ".stage-"+key)); err != nil {
		t.Fatal(err)
	}
	if err := s.Install(context.Background(), testSpec(), true, fetchTestModel); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal("followed staging symlink")
	}
	if err := os.Symlink(out, filepath.Join(s.Root, key, "unexpected")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(context.Background(), testSpec()); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("unsafe tree accepted: %v", err)
	}
}

func TestCommitAndRestoreFailuresPreserveRecoverableState(t *testing.T) {
	for _, failRestore := range []bool{false, true} {
		s := testStore(t)
		install(t, s)
		corrupt(t, s)
		key, _ := CacheKey(testSpec())
		s.operations = &storeOperations{rename: func(from, to string) error {
			if from == filepath.Join(s.Root, ".stage-"+key) || (failRestore && from == filepath.Join(s.Root, ".previous-"+key)) {
				return errors.New("injected rename failure")
			}
			return os.Rename(from, to)
		}}
		err := s.Install(context.Background(), testSpec(), true, fetchTestModel)
		if err == nil {
			t.Fatal("rename failure reported success")
		}
		if failRestore && !strings.Contains(err.Error(), "cannot restore") {
			t.Fatal("restore failure lost")
		}
		s.operations = nil
		if err := s.Install(context.Background(), testSpec(), false, fetchTestModel); !errors.Is(err, ErrCorrupt) {
			t.Fatalf("recovery did not preserve old content: %v", err)
		}
		if err := s.Install(context.Background(), testSpec(), true, fetchTestModel); err != nil {
			t.Fatal(err)
		}
		requireReady(t, s)
	}
}

func TestSyncFailureIsNotSuccessAndKeepsPreviousUntilRecovery(t *testing.T) {
	s := testStore(t)
	install(t, s)
	corrupt(t, s)
	key, _ := CacheKey(testSpec())
	committed := false
	s.operations = &storeOperations{rename: func(from, to string) error {
		err := os.Rename(from, to)
		if err == nil && from == filepath.Join(s.Root, ".stage-"+key) {
			committed = true
		}
		return err
	}, syncDirectory: func(path string) error {
		if committed && path == s.Root {
			return errors.New("injected directory sync failure")
		}
		return syncDirectory(path)
	}}
	if err := s.Install(context.Background(), testSpec(), true, fetchTestModel); err == nil {
		t.Fatal("sync failure reported success")
	}
	if _, err := os.Stat(filepath.Join(s.Root, ".previous-"+key)); err != nil {
		t.Fatal("old directory removed before sync")
	}
	s.operations = nil
	if err := s.Install(context.Background(), testSpec(), false, fetchTestModel); err != nil {
		t.Fatal(err)
	}
	requireReady(t, s)
}

func TestDifferentCacheKeysProgressIndependently(t *testing.T) {
	s := testStore(t)
	install(t, s)
	key, _ := CacheKey(testSpec())
	release, err := s.lock(context.Background(), key, true)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	other := testSpec()
	other.Repo = "owner/other"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Install(ctx, other, false, fetchTestModel); err != nil {
		t.Fatal(err)
	}
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("test subprocess did not reach checkpoint")
}

func modelProcess(t *testing.T, root, mode string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGGUFStoreSubprocess$")
	cmd.Env = append(os.Environ(), "COMMITER_GGUF_TEST_ROOT="+root, "COMMITER_GGUF_TEST_MODE="+mode)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestConcurrentSetupAcrossProcessesFetchesOnce(t *testing.T) {
	s := testStore(t)
	if err := os.Mkdir(s.Root, 0700); err != nil {
		t.Fatal(err)
	}
	first := modelProcess(t, s.Root, "concurrent")
	waitFile(t, filepath.Join(s.Root, "fetch.marker"))
	second := modelProcess(t, s.Root, "concurrent")
	if err := os.WriteFile(filepath.Join(s.Root, "release.marker"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := second.Wait(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(s.Root, "fetch.marker"))
	if err != nil || string(data) != "fetch\n" {
		t.Fatalf("fetched more than once: %q %v", data, err)
	}
	requireReady(t, s)
}

func TestKilledSetupRecoversAtEachCommitBoundary(t *testing.T) {
	for _, mode := range []string{"download", "metadata", "retired", "committed"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			if mode == "retired" || mode == "committed" {
				install(t, s)
				corrupt(t, s)
			} else if err := os.Mkdir(s.Root, 0700); err != nil {
				t.Fatal(err)
			}
			cmd := modelProcess(t, s.Root, mode)
			waitFile(t, filepath.Join(s.Root, "checkpoint.marker"))
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			lease, err := s.Open(context.Background(), testSpec())
			if mode == "committed" {
				if err != nil {
					t.Fatal(err)
				}
				_ = lease.Close()
			} else if !errors.Is(err, ErrIncomplete) {
				t.Fatalf("partial model readable or wrong state: %v", err)
			}
			if mode == "retired" {
				if err := s.Install(context.Background(), testSpec(), false, fetchTestModel); !errors.Is(err, ErrCorrupt) {
					t.Fatalf("failed to restore old corrupt cache: %v", err)
				}
			}
			if err := s.Install(context.Background(), testSpec(), true, fetchTestModel); err != nil {
				t.Fatal(err)
			}
			requireReady(t, s)
		})
	}
}

type checkpointReader struct {
	root  string
	reads atomic.Int32
}

func (r *checkpointReader) Read(p []byte) (int, error) {
	if r.reads.Add(1) == 1 {
		return copy(p, []byte(testModel[:4])), nil
	}
	_ = os.WriteFile(filepath.Join(r.root, "checkpoint.marker"), nil, 0600)
	for {
		time.Sleep(time.Second)
	}
}
func (*checkpointReader) Close() error { return nil }

// This helper uses synthetic bytes only. A real llama-server is never started.
func TestGGUFStoreSubprocess(t *testing.T) {
	root, mode := os.Getenv("COMMITER_GGUF_TEST_ROOT"), os.Getenv("COMMITER_GGUF_TEST_MODE")
	if root == "" || mode == "" {
		return
	}
	s := Store{Root: root}
	key, _ := CacheKey(testSpec())
	checkpoint := func() {
		if err := os.WriteFile(filepath.Join(root, "checkpoint.marker"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		for {
			time.Sleep(time.Second)
		}
	}
	s.operations = &storeOperations{
		rename: func(from, to string) error {
			if err := os.Rename(from, to); err != nil {
				return err
			}
			if mode == "retired" && to == filepath.Join(root, ".previous-"+key) || mode == "committed" && from == filepath.Join(root, ".stage-"+key) {
				checkpoint()
			}
			return nil
		},
		syncDirectory: func(path string) error {
			if mode == "metadata" && path == filepath.Join(root, ".stage-"+key) {
				if _, err := os.Stat(filepath.Join(path, metadataName)); err == nil {
					checkpoint()
				}
			}
			return syncDirectory(path)
		},
	}
	fetch := func(ctx context.Context, spec Spec, f File) (io.ReadCloser, error) {
		if mode == "download" {
			return &checkpointReader{root: root}, nil
		}
		if mode == "concurrent" {
			file, err := os.OpenFile(filepath.Join(root, "fetch.marker"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			_, _ = file.WriteString("fetch\n")
			_ = file.Close()
			waitFile(t, filepath.Join(root, "release.marker"))
		}
		return fetchTestModel(ctx, spec, f)
	}
	if err := s.Install(context.Background(), testSpec(), mode == "retired" || mode == "committed", fetch); err != nil {
		t.Fatal(err)
	}
}
