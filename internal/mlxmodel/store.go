// Package mlxmodel manages explicitly downloaded, revision-pinned MLX models.
// Normal inference uses Ready only; all registry I/O is confined to Plan and Install.
package mlxmodel

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxTreeBytes = 4 << 20
const maxModelBytes int64 = 50 << 30

type Spec struct {
	Repo         string `json:"repo"`
	Revision     string `json:"revision"`
	Quantization string `json:"quantization"`
}

type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
	LFS  bool   `json:"lfs"`
}

type Plan struct {
	Spec  Spec   `json:"spec"`
	Files []File `json:"files"`
	Bytes int64  `json:"bytes"`
}

type Store struct {
	Root string
	// BaseURL and Client are injectable for protocol tests. Production uses
	// the public Hugging Face endpoint and sends no auth or repository data.
	BaseURL string
	Client  *http.Client
}

func DefaultStore() (Store, error) {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		return Store{}, errors.New("MLX model cache location is unavailable")
	}
	return Store{Root: filepath.Join(root, "commiter", "mlx-models")}, nil
}

func (store Store) endpoint() string {
	if store.BaseURL != "" {
		return strings.TrimRight(store.BaseURL, "/")
	}
	return "https://huggingface.co"
}

func (store Store) client() *http.Client {
	if store.Client != nil {
		return store.Client
	}
	return &http.Client{Timeout: 45 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		host := strings.ToLower(req.URL.Hostname())
		if len(via) >= 5 || req.URL.Scheme != "https" ||
			(host != "huggingface.co" && !strings.HasSuffix(host, ".huggingface.co") &&
				host != "hf.co" && !strings.HasSuffix(host, ".hf.co")) {
			return errors.New("model download redirected outside the registry")
		}
		return nil
	}}
}

func Validate(spec Spec) error {
	parts := strings.Split(spec.Repo, "/")
	if len(parts) != 2 {
		return errors.New("MLX model repository must be owner/name")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return errors.New("invalid MLX model repository")
		}
		for _, char := range part {
			if !asciiNameChar(char) {
				return errors.New("invalid MLX model repository")
			}
		}
	}
	if len(spec.Revision) != 40 {
		return errors.New("MLX model revision must be a full commit hash")
	}
	for _, char := range spec.Revision {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return errors.New("MLX model revision must be a full commit hash")
		}
	}
	if spec.Quantization != "none" {
		if !strings.HasSuffix(spec.Quantization, "bit") {
			return errors.New("invalid MLX model quantization label")
		}
		label := strings.TrimSuffix(spec.Quantization, "bit")
		bits, err := strconv.Atoi(label)
		if err != nil || bits < 1 || bits > 99 || len(label) > 2 || label[0] == '0' {
			return errors.New("invalid MLX model quantization label")
		}
	}
	return nil
}

func asciiNameChar(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.'
}

func (store Store) directory(spec Spec) string {
	digest := sha256.Sum256([]byte(spec.Repo + "\n" + strings.ToLower(spec.Revision) + "\n" + spec.Quantization))
	return filepath.Join(store.Root, hex.EncodeToString(digest[:]))
}

func (store Store) Destination(spec Spec) (string, error) {
	if err := Validate(spec); err != nil {
		return "", err
	}
	return store.directory(spec), nil
}

func (store Store) Ready(spec Spec) (string, error) {
	if err := Validate(spec); err != nil {
		return "", err
	}
	dir := store.directory(spec)
	metadata, err := os.Open(filepath.Join(dir, "commiter-model.json"))
	if err != nil {
		return "", errors.New("MLX model is not installed; run commiter setup")
	}
	defer metadata.Close()
	data, err := io.ReadAll(io.LimitReader(metadata, maxTreeBytes+1))
	if err != nil {
		return "", errors.New("MLX model metadata is invalid")
	}
	var plan Plan
	if len(data) > maxTreeBytes || json.Unmarshal(data, &plan) != nil || plan.Spec != spec || len(plan.Files) == 0 {
		return "", errors.New("MLX model metadata is invalid; run commiter setup --update-model")
	}
	for _, file := range plan.Files {
		if !safePath(file.Path) || file.Size < 0 {
			return "", errors.New("MLX model metadata is invalid")
		}
		info, statErr := os.Lstat(filepath.Join(dir, filepath.FromSlash(file.Path)))
		if statErr != nil || !info.Mode().IsRegular() || info.Size() != file.Size {
			return "", errors.New("MLX model files are incomplete; run commiter setup --update-model")
		}
	}
	return dir, nil
}

func (store Store) Plan(ctx context.Context, spec Spec) (Plan, error) {
	if err := Validate(spec); err != nil {
		return Plan{}, err
	}
	endpoint := store.endpoint() + "/api/models/" + escapeRepo(spec.Repo) + "/tree/" + url.PathEscape(spec.Revision) + "?recursive=true&expand=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Plan{}, errors.New("cannot prepare model metadata request")
	}
	response, err := store.client().Do(req)
	if err != nil {
		return Plan{}, errors.New("cannot fetch MLX model metadata")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Plan{}, fmt.Errorf("MLX model metadata request failed (HTTP %d)", response.StatusCode)
	}
	if response.Header.Get("Link") != "" {
		return Plan{}, errors.New("MLX model tree is paginated and cannot be installed safely")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxTreeBytes+1))
	if err != nil || len(data) > maxTreeBytes {
		return Plan{}, errors.New("MLX model metadata is too large")
	}
	var entries []struct {
		Type string `json:"type"`
		Path string `json:"path"`
		Size int64  `json:"size"`
		OID  string `json:"oid"`
		LFS  *struct {
			OID string `json:"oid"`
		} `json:"lfs"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return Plan{}, errors.New("invalid MLX model metadata")
	}
	plan := Plan{Spec: spec}
	seen := make(map[string]bool)
	weights, config := false, false
	for _, entry := range entries {
		if entry.Type != "file" || !modelFile(entry.Path) {
			continue
		}
		if !safePath(entry.Path) || seen[entry.Path] || entry.Size < 0 || entry.Size > maxModelBytes-plan.Bytes {
			return Plan{}, errors.New("invalid MLX model file metadata")
		}
		seen[entry.Path] = true
		file := File{Path: entry.Path, Size: entry.Size, Hash: entry.OID}
		if entry.LFS != nil {
			file.LFS, file.Hash = true, entry.LFS.OID
		}
		wantLength := 40
		if file.LFS {
			wantLength = 64
		}
		if !hexHash(file.Hash, wantLength) {
			return Plan{}, errors.New("MLX model file has no verifiable digest")
		}
		plan.Files = append(plan.Files, file)
		plan.Bytes += file.Size
		weights = weights || strings.HasSuffix(file.Path, ".safetensors")
		config = config || file.Path == "config.json"
		if len(plan.Files) > 10000 {
			return Plan{}, errors.New("MLX model has too many files")
		}
	}
	if !weights || !config {
		return Plan{}, errors.New("MLX model must contain config.json and safetensors weights")
	}
	return plan, nil
}

func modelFile(path string) bool {
	for _, suffix := range []string{".json", ".safetensors", ".model", ".txt", ".jinja", ".tiktoken", ".vocab", ".merges"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func safePath(path string) bool {
	if path == "" || path == "commiter-model.json" || strings.Contains(path, "\\") || strings.ContainsAny(path, "\r\n") || filepath.IsAbs(path) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}

func hexHash(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func escapeRepo(repo string) string {
	parts := strings.Split(repo, "/")
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1])
}

func (store Store) Install(ctx context.Context, plan Plan) (string, error) {
	if err := Validate(plan.Spec); err != nil {
		return "", err
	}
	if len(plan.Files) == 0 {
		return "", errors.New("MLX model download plan is empty")
	}
	if ready, err := store.Ready(plan.Spec); err == nil {
		return ready, nil
	}
	if err := os.MkdirAll(store.Root, 0o700); err != nil {
		return "", errors.New("cannot create MLX model cache")
	}
	staging, err := os.MkdirTemp(store.Root, ".download-")
	if err != nil {
		return "", errors.New("cannot create MLX model staging directory")
	}
	defer os.RemoveAll(staging)
	var configFile *File
	for index := range plan.Files {
		if plan.Files[index].Path == "config.json" {
			configFile = &plan.Files[index]
			break
		}
	}
	if configFile == nil {
		return "", errors.New("MLX model download plan has no config.json")
	}
	if !safePath(configFile.Path) || configFile.Size < 0 || !hexHash(configFile.Hash, map[bool]int{true: 64, false: 40}[configFile.LFS]) {
		return "", errors.New("invalid MLX model download plan")
	}
	if err := downloadPlannedFile(ctx, store, plan.Spec, *configFile, staging); err != nil {
		return "", err
	}
	if err := verifyQuantization(filepath.Join(staging, configFile.Path), plan.Spec.Quantization); err != nil {
		return "", err
	}
	for _, file := range plan.Files {
		if file.Path == configFile.Path {
			continue
		}
		if !safePath(file.Path) || file.Size < 0 || !hexHash(file.Hash, map[bool]int{true: 64, false: 40}[file.LFS]) {
			return "", errors.New("invalid MLX model download plan")
		}
		if err := downloadPlannedFile(ctx, store, plan.Spec, file, staging); err != nil {
			return "", err
		}
	}
	metadata, err := json.Marshal(plan)
	if err != nil || os.WriteFile(filepath.Join(staging, "commiter-model.json"), metadata, 0o600) != nil {
		return "", errors.New("cannot write MLX model metadata")
	}
	destination := store.directory(plan.Spec)
	backup := staging + "-previous"
	if _, statErr := os.Lstat(destination); statErr == nil {
		if err := os.Rename(destination, backup); err != nil {
			return "", errors.New("cannot replace incomplete MLX model")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", errors.New("cannot inspect MLX model destination")
	}
	if err := os.Rename(staging, destination); err != nil {
		if _, backupErr := os.Lstat(backup); backupErr == nil {
			_ = os.Rename(backup, destination)
		}
		if ready, readyErr := store.Ready(plan.Spec); readyErr == nil {
			return ready, nil
		}
		return "", errors.New("cannot publish MLX model; an incomplete destination may exist")
	}
	_ = os.RemoveAll(backup)
	return destination, nil
}

func verifyQuantization(path, label string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.New("MLX model configuration is unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return errors.New("MLX model configuration is unavailable")
	}
	var modelConfig struct {
		Quantization *struct {
			Bits int `json:"bits"`
		} `json:"quantization"`
	}
	if json.Unmarshal(data, &modelConfig) != nil {
		return errors.New("MLX model configuration is invalid")
	}
	if modelConfig.Quantization == nil && label == "none" {
		return nil
	}
	if modelConfig.Quantization != nil && strings.HasSuffix(label, "bit") {
		bits, parseErr := strconv.Atoi(strings.TrimSuffix(label, "bit"))
		if parseErr == nil && bits > 0 && bits == modelConfig.Quantization.Bits {
			return nil
		}
	}
	return errors.New("MLX model quantization does not match pinned configuration")
}

func downloadPlannedFile(ctx context.Context, store Store, spec Spec, file File, staging string) error {
	segments := strings.Split(file.Path, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	endpoint := store.endpoint() + "/" + escapeRepo(spec.Repo) + "/resolve/" + url.PathEscape(spec.Revision) + "/" + strings.Join(segments, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return errors.New("cannot prepare MLX model download")
	}
	response, err := store.client().Do(req)
	if err != nil {
		return errors.New("MLX model download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("MLX model download failed (HTTP %d)", response.StatusCode)
	}
	path := filepath.Join(staging, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errors.New("cannot create MLX model directory")
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.New("cannot create MLX model file")
	}
	hash256 := sha256.New()
	hash1 := sha1.New()
	_, _ = fmt.Fprintf(hash1, "blob %d\x00", file.Size)
	count, copyErr := io.Copy(io.MultiWriter(out, hash256, hash1), io.LimitReader(response.Body, file.Size+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || count != file.Size {
		return errors.New("MLX model download has an incomplete file")
	}
	actual := hex.EncodeToString(hash1.Sum(nil))
	if file.LFS {
		actual = hex.EncodeToString(hash256.Sum(nil))
	}
	if !strings.EqualFold(actual, file.Hash) {
		return errors.New("MLX model download digest mismatch")
	}
	return nil
}
