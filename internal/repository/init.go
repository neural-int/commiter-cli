package repository

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InitializationPlan describes the changes that initialization would make.
type InitializationPlan struct {
	Root            string
	InitializeGit   bool
	CreateGitignore bool
}

// PlanInitialization detects the repository boundary without changing it.
func PlanInitialization(path string) (InitializationPlan, error) {
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
	_, err = os.Stat(filepath.Join(root, ".gitignore"))
	createGitignore := errors.Is(err, os.ErrNotExist)
	if err != nil && !createGitignore {
		return InitializationPlan{}, fmt.Errorf("cannot inspect .gitignore")
	}
	return InitializationPlan{Root: root, InitializeGit: !found, CreateGitignore: createGitignore}, nil
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
	current, err := PlanInitialization(plan.Root)
	if err != nil {
		return err
	}
	if current.InitializeGit != plan.InitializeGit {
		return fmt.Errorf("repository state changed after confirmation; rerun init")
	}
	plan.CreateGitignore = plan.CreateGitignore && current.CreateGitignore
	if plan.InitializeGit {
		command := exec.Command("git", "init", plan.Root)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("cannot initialize Git repository: %s", strings.TrimSpace(string(output)))
		}
	}
	if plan.CreateGitignore {
		file, err := os.OpenFile(filepath.Join(plan.Root, ".gitignore"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot create .gitignore")
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("cannot close .gitignore")
		}
	}
	return nil
}

func detectRoot(path string) (string, bool, error) {
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
