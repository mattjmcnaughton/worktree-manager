package resolver

import (
	"path/filepath"
	"testing"

	"github.com/mattjmcnaughton/worktree-manager/internal/config"
)

func TestResolveAppliesTemplatesAndPaths(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Base:             "main",
			WorktreeBaseDir:  ".worktrees",
			BranchTemplate:   "{{ user }}/{{ slug }}",
			WorktreeTemplate: "{{ repo }}-{{ slug }}",
			User:             "matt",
		},
		Agentic: config.Agentic{Enabled: true, WorkspaceDir: ".agentic", CreateTaskWorkspace: true},
	}
	in := Inputs{
		Slug:     "add-semantic-indexing",
		RepoName: "worktree-manager",
		RepoRoot: "/repo",
	}
	got, err := Resolve(cfg, in)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Branch != "matt/add-semantic-indexing" {
		t.Errorf("Branch = %q", got.Branch)
	}
	wantPath := filepath.Join("/repo", ".worktrees", "worktree-manager-add-semantic-indexing")
	if got.WorktreePath != wantPath {
		t.Errorf("WorktreePath = %q, want %q", got.WorktreePath, wantPath)
	}
	wantWS := filepath.Join("/repo", ".agentic", "add-semantic-indexing")
	if got.WorkspacePath != wantWS {
		t.Errorf("WorkspacePath = %q, want %q", got.WorkspacePath, wantWS)
	}
	if got.Base != "main" {
		t.Errorf("Base = %q", got.Base)
	}
}

func TestResolveOverridesBase(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Base: "main", WorktreeBaseDir: ".wt",
			BranchTemplate: "{{ slug }}", WorktreeTemplate: "{{ slug }}", User: "u",
		},
	}
	got, err := Resolve(cfg, Inputs{Slug: "x", RepoName: "r", RepoRoot: "/r", BaseOverride: "develop"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Base != "develop" {
		t.Errorf("Base = %q, want develop", got.Base)
	}
}

func TestResolveWorkspaceDisabledWhenNotConfigured(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Base: "main", WorktreeBaseDir: ".wt",
			BranchTemplate: "{{ slug }}", WorktreeTemplate: "{{ slug }}", User: "u",
		},
		Agentic: config.Agentic{Enabled: false, WorkspaceDir: ".agentic"},
	}
	got, err := Resolve(cfg, Inputs{Slug: "x", RepoName: "r", RepoRoot: "/r"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.WorkspacePath != "" {
		t.Errorf("WorkspacePath should be empty when agentic disabled; got %q", got.WorkspacePath)
	}
}

func TestResolveCreateWorkspaceFlagForcesEnabled(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			Base: "main", WorktreeBaseDir: ".wt",
			BranchTemplate: "{{ slug }}", WorktreeTemplate: "{{ slug }}", User: "u",
		},
		Agentic: config.Agentic{Enabled: true, WorkspaceDir: ".agentic", CreateTaskWorkspace: false},
	}
	got, err := Resolve(cfg, Inputs{Slug: "x", RepoName: "r", RepoRoot: "/r", AgenticOverride: ptrBool(true)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.WorkspacePath == "" {
		t.Errorf("expected WorkspacePath set when AgenticOverride=true")
	}
}

func ptrBool(b bool) *bool { return &b }
