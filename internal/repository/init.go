package repository

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InitializationPlan describes the changes that initialization would make.
type InitializationPlan struct {
	Root             string
	InitializeGit    bool
	CreateGitignore  bool
	UpdateGitignore  bool
	GitignoreBefore  []byte
	GitignoreExists  bool
	GitignoreEntries []string
}

// PlanInitialization detects the repository boundary without changing it.
func PlanInitialization(path string) (InitializationPlan, error) {
	return PlanInitializationWithIgnore(path, nil)
}

// PlanInitializationWithIgnore plans initialization and appends the explicitly requested
// patterns to .gitignore when they are not already present.
func PlanInitializationWithIgnore(path string, patterns []string) (InitializationPlan, error) {
	patterns, err := normalizeGitignorePatterns(patterns)
	if err != nil {
		return InitializationPlan{}, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return InitializationPlan{}, fmt.Errorf("cannot resolve initialization directory")
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return InitializationPlan{}, fmt.Errorf("initialization directory does not exist")
	}

	root, found, err := detectRoot(abs)
	if err != nil {
		return InitializationPlan{}, err
	}
	if !found {
		if metadataPath, metadataFound, statErr := findGitMetadata(abs); statErr != nil {
			return InitializationPlan{}, statErr
		} else if metadataFound {
			return InitializationPlan{}, fmt.Errorf("existing .git metadata at %s is invalid; refusing to initialize", metadataPath)
		}
		root = abs
	}
	gitignorePath := filepath.Join(root, ".gitignore")
	if info, statErr := os.Lstat(gitignorePath); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return InitializationPlan{}, fmt.Errorf(".gitignore is a symbolic link; refusing to modify it")
	}
	gitignoreBefore, err := os.ReadFile(gitignorePath)
	gitignoreExists := err == nil
	createGitignore := errors.Is(err, os.ErrNotExist)
	if err != nil && !createGitignore {
		return InitializationPlan{}, fmt.Errorf("cannot inspect .gitignore")
	}
	missing := missingGitignorePatterns(gitignoreBefore, patterns)
	return InitializationPlan{
		Root: root, InitializeGit: !found, CreateGitignore: createGitignore,
		UpdateGitignore: createGitignore && len(patterns) > 0 || len(missing) > 0,
		GitignoreBefore: gitignoreBefore, GitignoreExists: gitignoreExists,
		GitignoreEntries: missing,
	}, nil
}

func normalizeGitignorePatterns(patterns []string) ([]string, error) {
	seen := make(map[string]bool, len(patterns))
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "\r\n") {
			return nil, fmt.Errorf("gitignore pattern must be a single line")
		}
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return nil, fmt.Errorf("gitignore pattern must not be empty")
		}
		if !seen[pattern] {
			seen[pattern] = true
			result = append(result, pattern)
		}
	}
	return result, nil
}

func missingGitignorePatterns(content []byte, patterns []string) []string {
	existing := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		existing[strings.TrimSuffix(line, "\r")] = true
	}
	missing := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if !existing[pattern] {
			missing = append(missing, pattern)
		}
	}
	return missing
}

func findGitMetadata(path string) (string, bool, error) {
	current := path
	for {
		metadataPath := filepath.Join(current, ".git")
		if _, err := os.Lstat(metadataPath); err == nil {
			return metadataPath, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("cannot inspect .git metadata")
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}

// ApplyInitialization performs a previously displayed initialization plan.
func ApplyInitialization(plan InitializationPlan) error {
	current, err := PlanInitializationWithIgnore(plan.Root, plan.GitignoreEntries)
	if err != nil {
		return err
	}
	if current.InitializeGit != plan.InitializeGit {
		return fmt.Errorf("repository state changed after confirmation; rerun init")
	}
	if current.GitignoreExists != plan.GitignoreExists || !bytes.Equal(current.GitignoreBefore, plan.GitignoreBefore) {
		return fmt.Errorf(".gitignore changed after confirmation; rerun init")
	}
	if plan.InitializeGit {
		command := exec.Command("git", "init", plan.Root)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("cannot initialize Git repository: %s", strings.TrimSpace(string(output)))
		}
	}
	if plan.CreateGitignore || plan.UpdateGitignore {
		gitignorePath := filepath.Join(plan.Root, ".gitignore")
		if plan.UpdateGitignore && plan.GitignoreExists {
			file, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0o644)
			if err != nil {
				return fmt.Errorf("cannot open .gitignore")
			}
			content, readErr := io.ReadAll(file)
			if readErr != nil || !bytes.Equal(content, plan.GitignoreBefore) {
				_ = file.Close()
				return fmt.Errorf(".gitignore changed after confirmation; rerun init")
			}
			addition := []byte(strings.Join(plan.GitignoreEntries, "\n"))
			if len(content) > 0 && content[len(content)-1] != '\n' {
				addition = append([]byte{'\n'}, addition...)
			}
			addition = append(addition, '\n')
			if _, err := file.Seek(0, io.SeekEnd); err != nil {
				_ = file.Close()
				return fmt.Errorf("cannot seek .gitignore")
			}
			if _, err := file.Write(addition); err != nil {
				_ = file.Close()
				return fmt.Errorf("cannot write .gitignore")
			}
			return file.Close()
		}
		content := plan.GitignoreBefore
		if plan.UpdateGitignore {
			content = append(content, []byte(strings.Join(plan.GitignoreEntries, "\n"))...)
			content = append(content, '\n')
		}
		flags := os.O_WRONLY | os.O_CREATE
		if plan.GitignoreExists {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_EXCL
		}
		file, err := os.OpenFile(gitignorePath, flags, 0o644)
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot create .gitignore")
		}
		if _, err := file.Write(content); err != nil {
			_ = file.Close()
			return fmt.Errorf("cannot write .gitignore")
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("cannot close .gitignore")
		}
	}
	return nil
}

func detectRoot(path string) (string, bool, error) {
	bareCommand := exec.Command("git", "-C", path, "rev-parse", "--is-bare-repository")
	var bareStdout bytes.Buffer
	bareCommand.Stdout = &bareStdout
	if err := bareCommand.Run(); err == nil && strings.TrimSpace(bareStdout.String()) == "true" {
		return "", false, fmt.Errorf("bare Git repository at %s is not supported; refusing to initialize", path)
	}

	command := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil {
		return "", false, nil
	}
	root, err := filepath.Abs(strings.TrimSpace(stdout.String()))
	if err != nil {
		return "", false, fmt.Errorf("cannot resolve repository root")
	}
	return root, true, nil
}
