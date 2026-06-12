package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsWhenBothFilesMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "global.yml"), filepath.Join(dir, "repo.yml"), "")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.Base != "main" {
		t.Errorf("Defaults.Base = %q, want %q", cfg.Defaults.Base, "main")
	}
	if cfg.Defaults.WorktreeBaseDir != ".worktrees" {
		t.Errorf("Defaults.WorktreeBaseDir = %q, want %q", cfg.Defaults.WorktreeBaseDir, ".worktrees")
	}
	if cfg.Defaults.BranchTemplate != "{{ user }}/{{ slug }}" {
		t.Errorf("Defaults.BranchTemplate = %q, want %q", cfg.Defaults.BranchTemplate, "{{ user }}/{{ slug }}")
	}
	if cfg.Defaults.WorktreeTemplate != "{{ repo }}-{{ slug }}" {
		t.Errorf("Defaults.WorktreeTemplate = %q, want %q", cfg.Defaults.WorktreeTemplate, "{{ repo }}-{{ slug }}")
	}
	if cfg.Agentic.WorkspaceDir != ".agentic" {
		t.Errorf("Agentic.WorkspaceDir = %q, want %q", cfg.Agentic.WorkspaceDir, ".agentic")
	}
}

func TestLoadRepoOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yml")
	repoPath := filepath.Join(dir, "repo.yml")

	writeFile(t, globalPath, `version: "1.0"
defaults:
  base: develop
  worktree_base_dir: ".worktrees"
agentic:
  enabled: false
`)
	writeFile(t, repoPath, `version: "1.0"
defaults:
  base: main
agentic:
  enabled: true
  create_task_workspace: true
`)

	cfg, err := Load(globalPath, repoPath, "")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.Base != "main" {
		t.Errorf("Defaults.Base = %q, want %q", cfg.Defaults.Base, "main")
	}
	if cfg.Defaults.WorktreeBaseDir != ".worktrees" {
		t.Errorf("Defaults.WorktreeBaseDir = %q, want inherited %q", cfg.Defaults.WorktreeBaseDir, ".worktrees")
	}
	if !cfg.Agentic.Enabled {
		t.Errorf("Agentic.Enabled = false, want true (repo override)")
	}
	if !cfg.Agentic.CreateTaskWorkspace {
		t.Errorf("Agentic.CreateTaskWorkspace = false, want true (repo override)")
	}
}

func TestValidateRejectsBadHooks(t *testing.T) {
	cases := []struct {
		name string
		hook Hook
	}{
		{"copy missing from", Hook{Type: "copy"}},
		{"command missing command", Hook{Type: "command"}},
		{"symlink missing to", Hook{Type: "symlink", From: "a"}},
		{"symlink missing from", Hook{Type: "symlink", To: "b"}},
		{"unknown type", Hook{Type: "explode"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Hooks: Hooks{PostCreate: []Hook{tc.hook}}}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestValidateAcceptsGoodHooks(t *testing.T) {
	cfg := &Config{Hooks: Hooks{
		PreCreate:  []Hook{{Type: "command", Command: "echo pre"}},
		PostCreate: []Hook{{Type: "copy", From: ".env", To: ".env"}, {Type: "symlink", From: ".cache", To: ".cache"}},
		PreDelete:  []Hook{{Type: "command", Command: "git status"}},
		PostDelete: []Hook{{Type: "command", Command: "git worktree prune"}},
	}}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config to pass; got %v", err)
	}
}

func TestLoadParsesWtpConfig(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, ".wtp.yml")
	writeFile(t, repoPath, `version: "1.0"
defaults:
  base_dir: "../worktrees"
hooks:
  post_create:
    - type: copy
      from: ".env"
      to: ".env"
    - type: copy
      from: ".claude"
    - type: symlink
      from: ".bin"
      to: ".bin"
    - type: command
      command: "npm install"
      env:
        NODE_ENV: "development"
`)

	cfg, err := Load("", repoPath, "")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.WorktreeBaseDir != "../worktrees" {
		t.Errorf("Defaults.WorktreeBaseDir = %q, want %q (from base_dir alias)", cfg.Defaults.WorktreeBaseDir, "../worktrees")
	}
	if got := len(cfg.Hooks.PostCreate); got != 4 {
		t.Fatalf("PostCreate hook count = %d, want 4", got)
	}
}

func TestDefaultsBaseDirAliasYieldsToCanonicalKey(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "repo.yml")
	writeFile(t, repoPath, `defaults:
  base_dir: "../wtp-style"
  worktree_base_dir: ".native"
`)

	cfg, err := Load("", repoPath, "")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.WorktreeBaseDir != ".native" {
		t.Errorf("Defaults.WorktreeBaseDir = %q, want %q (canonical key wins)", cfg.Defaults.WorktreeBaseDir, ".native")
	}
}

func TestLoadMergesPerRepoOverride(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repoRoot: %v", err)
	}
	globalPath := filepath.Join(dir, "global.yml")

	writeFile(t, globalPath, `version: "1.0"
defaults:
  base: develop
  user: globaluser
  branch_template: "{{ user }}/{{ slug }}"
repos:
  `+repoRoot+`:
    defaults:
      user: matt
      base: feature
`)

	cfg, err := Load(globalPath, "", repoRoot)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.User != "matt" {
		t.Errorf("Defaults.User = %q, want %q (per-repo override)", cfg.Defaults.User, "matt")
	}
	if cfg.Defaults.Base != "feature" {
		t.Errorf("Defaults.Base = %q, want %q (per-repo override)", cfg.Defaults.Base, "feature")
	}
	if cfg.Defaults.BranchTemplate != "{{ user }}/{{ slug }}" {
		t.Errorf("Defaults.BranchTemplate = %q, want %q (inherited from global defaults)", cfg.Defaults.BranchTemplate, "{{ user }}/{{ slug }}")
	}
	if cfg.Repos != nil {
		t.Errorf("merged Config.Repos = %v, want nil (only meaningful pre-merge)", cfg.Repos)
	}
}

func TestLoadRepoConfigWinsOverPerRepoOverride(t *testing.T) {
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repoRoot: %v", err)
	}
	globalPath := filepath.Join(dir, "global.yml")
	repoPath := filepath.Join(repoRoot, ".worktree-manager.yml")

	writeFile(t, globalPath, `repos:
  `+repoRoot+`:
    defaults:
      user: from-override
      base: from-override
`)
	writeFile(t, repoPath, `defaults:
  user: from-repo
`)

	cfg, err := Load(globalPath, repoPath, repoRoot)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.User != "from-repo" {
		t.Errorf("Defaults.User = %q, want %q (repo file wins)", cfg.Defaults.User, "from-repo")
	}
	if cfg.Defaults.Base != "from-override" {
		t.Errorf("Defaults.Base = %q, want %q (override fills when repo file is silent)", cfg.Defaults.Base, "from-override")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
