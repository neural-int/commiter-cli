package ggufmodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrMissing    = errors.New("managed GGUF model is not installed; run setup after runtime qualification")
	ErrIncomplete = errors.New("managed GGUF model is incomplete; run setup to recover")
	ErrCorrupt    = errors.New("managed GGUF model is corrupt; use setup --update-model to repair")
	ErrUnsafe     = errors.New("GGUF cache has an unsafe owner, permission or file type")
	ErrConflict   = errors.New("GGUF cache recovery state is contradictory; no files were changed")
	ErrBusy       = errors.New("GGUF cache lock is busy")
)

type Store struct {
	Root       string
	operations *storeOperations
}

type storeOperations struct {
	rename        func(string, string) error
	syncDirectory func(string) error
}

func (s Store) rename(from, to string) error {
	if s.operations != nil && s.operations.rename != nil {
		return s.operations.rename(from, to)
	}
	return os.Rename(from, to)
}

func (s Store) sync(path string) error {
	if s.operations != nil && s.operations.syncDirectory != nil {
		return s.operations.syncDirectory(path)
	}
	return syncDirectory(path)
}

// Fetch opens an explicitly requested pinned artifact. Store never performs
// network I/O during Open, and does not accept an external inference endpoint.
type Fetch func(context.Context, Spec, File) (io.ReadCloser, error)

type Lease struct {
	Path    string
	release func() error
}

func (l *Lease) Close() error { return l.release() }

type metadata struct {
	Version string `json:"version"`
	Runtime string `json:"runtime"`
	Key     string `json:"key"`
	Spec    Spec   `json:"spec"`
}

func secureDirectory(path string, create bool) error {
	if create {
		if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return ErrUnsafe
		}
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrMissing
	}
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || !owned(info) {
		return ErrUnsafe
	}
	return nil
}

func (s Store) lock(ctx context.Context, key string, exclusive bool) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(s.Root) {
		return nil, ErrUnsafe
	}
	if err := secureDirectory(s.Root, exclusive); err != nil {
		return nil, err
	}
	// Readers create a stable lock inode when necessary, but never repair a model.
	if err := secureDirectory(filepath.Join(s.Root, ".locks"), true); err != nil {
		return nil, err
	}
	return acquire(ctx, filepath.Join(s.Root, ".locks", key+".lock"), exclusive)
}

func (s Store) Open(ctx context.Context, spec Spec) (*Lease, error) {
	key, err := CacheKey(spec)
	if err != nil {
		return nil, err
	}
	release, err := s.lock(ctx, key, false)
	if err != nil {
		return nil, err
	}
	destination := filepath.Join(s.Root, key)
	if err := s.verify(ctx, destination, key, spec); err != nil {
		_ = release()
		if errors.Is(err, ErrMissing) {
			for _, name := range []string{".stage-", ".previous-"} {
				if _, statErr := os.Lstat(filepath.Join(s.Root, name+key)); statErr == nil {
					return nil, ErrIncomplete
				}
			}
		}
		return nil, err
	}
	return &Lease{Path: filepath.Join(destination, filepath.FromSlash(spec.Files[0].Path)), release: release}, nil
}

func inspectRegular(path string) (*os.File, error) {
	f, err := openNoFollow(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, ErrCorrupt
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !owned(info) {
		_ = f.Close()
		return nil, ErrUnsafe
	}
	return f, nil
}

func readContext(ctx context.Context, reader io.Reader, writer io.Writer) (int64, error) {
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			written, writeErr := writer.Write(buffer[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func (s Store) verify(ctx context.Context, directory, key string, spec Spec) error {
	if err := secureDirectory(directory, false); err != nil {
		return err
	}
	if err := inspectTree(directory); err != nil {
		return err
	}
	mf, err := inspectRegular(filepath.Join(directory, metadataName))
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(mf, 64<<10+1))
	closeErr := mf.Close()
	if readErr != nil || closeErr != nil || len(data) > 64<<10 {
		return ErrCorrupt
	}
	var found metadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&found); err != nil {
		return ErrCorrupt
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ErrCorrupt
	}
	expected, _ := json.Marshal(metadata{Version: formatVersion, Runtime: "llamacpp", Key: key, Spec: normalized(spec)})
	actual, _ := json.Marshal(found)
	if !bytes.Equal(expected, actual) {
		return ErrCorrupt
	}
	for _, artifact := range spec.Files {
		f, err := inspectRegular(filepath.Join(directory, filepath.FromSlash(artifact.Path)))
		if err != nil {
			return err
		}
		h := sha256.New()
		size, copyErr := readContext(ctx, io.LimitReader(f, artifact.Size+1), h)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil || size != artifact.Size || hex.EncodeToString(h.Sum(nil)) != normalized(spec).Files[0].SHA256 {
			return ErrCorrupt
		}
	}
	return nil
}

// inspectTree never follows symlinks, including unexpected nested entries.
func inspectTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return ErrUnsafe
		}
		info, err := entry.Info()
		if err != nil || !owned(info) {
			return ErrUnsafe
		}
		if info.IsDir() {
			if info.Mode().Perm() != 0o700 {
				return ErrUnsafe
			}
			return nil
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			return ErrUnsafe
		}
		return nil
	})
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.New("cannot open GGUF cache directory for sync")
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("cannot sync GGUF cache directory")
	}
	return nil
}

// removeOwnedTree only removes a validated, key-specific management directory.
func removeOwnedTree(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := secureDirectory(path, false); err != nil {
		return err
	}
	if err := inspectTree(path); err != nil {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return errors.New("cannot clean GGUF staging directory")
	}
	return nil
}

func (s Store) Install(ctx context.Context, spec Spec, update bool, fetch Fetch) (err error) {
	key, err := CacheKey(spec)
	if err != nil {
		return err
	}
	release, err := s.lock(ctx, key, true)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	destination, stage, previous := filepath.Join(s.Root, key), filepath.Join(s.Root, ".stage-"+key), filepath.Join(s.Root, ".previous-"+key)
	for _, path := range []string{destination, stage, previous} {
		if info, statErr := os.Lstat(path); statErr == nil {
			if !info.IsDir() || info.Mode().Perm() != 0o700 || !owned(info) {
				return ErrConflict
			}
			if err := inspectTree(path); err != nil {
				return ErrConflict
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return ErrConflict
		}
	}
	state := s.verify(ctx, destination, key, spec)
	_, previousErr := os.Lstat(previous)
	hasPrevious := previousErr == nil
	if state == nil {
		if err := s.sync(s.Root); err != nil {
			return err
		}
		if err := removeOwnedTree(stage); err != nil {
			return err
		}
		if err := removeOwnedTree(previous); err != nil {
			return err
		}
		return s.sync(s.Root)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if hasPrevious {
		if !errors.Is(state, ErrMissing) {
			return ErrConflict
		}
		if err := s.rename(previous, destination); err != nil {
			return errors.New("cannot restore previous GGUF cache")
		}
		if err := s.sync(s.Root); err != nil {
			return err
		}
		state = s.verify(ctx, destination, key, spec)
		if state == nil {
			if err := removeOwnedTree(stage); err != nil {
				return err
			}
			return s.sync(s.Root)
		}
	}
	if !errors.Is(state, ErrMissing) && !errors.Is(state, ErrCorrupt) {
		return state
	}
	if errors.Is(state, ErrCorrupt) && !update {
		return ErrCorrupt
	}
	if err := removeOwnedTree(stage); err != nil {
		return err
	}
	if fetch == nil {
		return errors.New("GGUF artifact fetcher is unavailable")
	}
	if err := secureDirectory(stage, true); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, removeOwnedTree(stage))
		}
	}()
	if err := s.writeStage(ctx, stage, key, spec, fetch); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	repair := errors.Is(state, ErrCorrupt)
	if repair {
		if err := s.rename(destination, previous); err != nil {
			return errors.New("cannot preserve previous GGUF cache")
		}
		if err := s.sync(s.Root); err != nil {
			return err
		}
	}
	if err := s.rename(stage, destination); err != nil {
		failure := errors.New("cannot commit GGUF cache")
		if repair {
			if restoreErr := s.rename(previous, destination); restoreErr != nil {
				return errors.Join(failure, errors.New("cannot restore previous GGUF cache; retry setup recovery"))
			}
			return errors.Join(failure, s.sync(s.Root))
		}
		return failure
	}
	if err := s.sync(s.Root); err != nil {
		return err
	}
	if err := removeOwnedTree(previous); err != nil {
		return err
	}
	return s.sync(s.Root)
}

func (s Store) writeStage(ctx context.Context, stage, key string, spec Spec, fetch Fetch) error {
	for _, artifact := range spec.Files {
		parent := filepath.Dir(filepath.Join(stage, filepath.FromSlash(artifact.Path)))
		// Stage is new and private. Create each registered path component without
		// accepting symlinks or reusing another process's staging directory.
		rel, _ := filepath.Rel(stage, parent)
		current := stage
		if rel != "." {
			for _, component := range splitPath(rel) {
				current = filepath.Join(current, component)
				if err := secureDirectory(current, true); err != nil {
					return err
				}
			}
		}
		reader, err := fetch(ctx, normalized(spec), artifact)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("cannot fetch pinned GGUF artifact")
		}
		file, err := openNoFollow(filepath.Join(stage, filepath.FromSlash(artifact.Path)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = reader.Close()
			return errors.New("cannot create GGUF staging file")
		}
		h := sha256.New()
		size, copyErr := readContext(ctx, io.LimitReader(reader, artifact.Size+1), io.MultiWriter(file, h))
		readCloseErr := reader.Close()
		syncErr := file.Sync()
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if readCloseErr != nil || syncErr != nil || closeErr != nil {
			return errors.New("cannot persist GGUF staging file")
		}
		if size != artifact.Size || hex.EncodeToString(h.Sum(nil)) != normalized(spec).Files[0].SHA256 {
			return ErrCorrupt
		}
		if err := s.sync(parent); err != nil {
			return err
		}
	}
	data, _ := json.Marshal(metadata{Version: formatVersion, Runtime: "llamacpp", Key: key, Spec: normalized(spec)})
	f, err := openNoFollow(filepath.Join(stage, metadataName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("cannot create GGUF metadata")
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("cannot persist GGUF metadata")
	}
	// Sync every created directory from the leaves up, including intermediate
	// artifact directories, before the atomic directory commit.
	var directories []string
	if err := filepath.WalkDir(stage, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return errors.New("cannot inspect GGUF staging directories")
		}
		if entry.IsDir() {
			directories = append(directories, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := s.sync(directories[i]); err != nil {
			return err
		}
	}
	return nil
}

func splitPath(path string) []string {
	if path == "." {
		return nil
	}
	parent, name := filepath.Split(path)
	return append(splitPath(filepath.Clean(parent)), name)
}
