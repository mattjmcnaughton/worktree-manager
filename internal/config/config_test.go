package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsWhenBothFilesMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "global.yml"), filepath.Join(dir, "repo.yml"))
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
	if cfg.Context.SourcesDir != ".agentic/sources" {
		t.Errorf("Context.SourcesDir = %q, want %q", cfg.Context.SourcesDir, ".agentic/sources")
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

	cfg, err := Load(globalPath, repoPath)
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

func TestLoadMergesContextSourcesByName(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yml")
	repoPath := filepath.Join(dir, "repo.yml")

	writeFile(t, globalPath, `version: "1.0"
context:
  enabled: true
  sources:
    - name: skills
      repo: mattjmcnaughton/skills
      ref: main
    - name: wtp
      repo: satococoa/wtp
      ref: main
`)
	writeFile(t, repoPath, `version: "1.0"
context:
  sources:
    - name: wtp
      ref: v0.5.0
    - name: extra
      repo: example/extra
`)

	cfg, err := Load(globalPath, repoPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	bySrc := make(map[string]ContextSource)
	for _, s := range cfg.Context.Sources {
		bySrc[s.Name] = s
	}

	if len(bySrc) != 3 {
		t.Fatalf("expected 3 merged sources, got %d (%v)", len(bySrc), bySrc)
	}
	if bySrc["skills"].Repo != "mattjmcnaughton/skills" {
		t.Errorf("skills.Repo not preserved: %+v", bySrc["skills"])
	}
	if bySrc["wtp"].Ref != "v0.5.0" {
		t.Errorf("wtp.Ref not overridden: %+v", bySrc["wtp"])
	}
	if bySrc["wtp"].Repo != "satococoa/wtp" {
		t.Errorf("wtp.Repo not inherited from global: %+v", bySrc["wtp"])
	}
	if bySrc["extra"].Repo != "example/extra" {
		t.Errorf("extra source not added: %+v", bySrc["extra"])
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
