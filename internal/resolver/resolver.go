// Package resolver turns a slug plus config into concrete paths and branch names.
package resolver

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mattjmcnaughton/worktree-manager/internal/config"
	"github.com/mattjmcnaughton/worktree-manager/internal/slug"
)

// Inputs collects everything required to resolve a managed task.
type Inputs struct {
	Slug            string
	RepoName        string
	RepoRoot        string
	BaseOverride    string
	AgenticOverride *bool
}

// Resolved is the output of Resolve.
type Resolved struct {
	Slug          string
	Branch        string
	Base          string
	WorktreePath  string
	WorkspacePath string
}

// Resolve renders templates and produces absolute paths.
func Resolve(cfg *config.Config, in Inputs) (Resolved, error) {
	if err := slug.Validate(in.Slug); err != nil {
		return Resolved{}, err
	}

	branchTpl := strings.NewReplacer("{{ ", "{{.").Replace(cfg.Defaults.BranchTemplate)
	branchTpl = strings.ReplaceAll(branchTpl, " }}", "}}")
	worktreeTpl := strings.NewReplacer("{{ ", "{{.").Replace(cfg.Defaults.WorktreeTemplate)
	worktreeTpl = strings.ReplaceAll(worktreeTpl, " }}", "}}")

	data := map[string]string{
		"user": cfg.Defaults.User,
		"slug": in.Slug,
		"repo": in.RepoName,
	}

	branch, err := render("branch_template", branchTpl, data)
	if err != nil {
		return Resolved{}, err
	}
	wtName, err := render("worktree_template", worktreeTpl, data)
	if err != nil {
		return Resolved{}, err
	}

	base := cfg.Defaults.Base
	if in.BaseOverride != "" {
		base = in.BaseOverride
	}

	worktreeBase := cfg.Defaults.WorktreeBaseDir
	if !filepath.IsAbs(worktreeBase) {
		worktreeBase = filepath.Join(in.RepoRoot, worktreeBase)
	}
	worktreePath := filepath.Join(worktreeBase, wtName)

	workspacePath := ""
	wantWorkspace := cfg.Agentic.Enabled && cfg.Agentic.CreateTaskWorkspace
	if in.AgenticOverride != nil {
		wantWorkspace = *in.AgenticOverride
	}
	if wantWorkspace {
		dir := cfg.Agentic.WorkspaceDir
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(in.RepoRoot, dir)
		}
		workspacePath = filepath.Join(dir, in.Slug)
	}

	return Resolved{
		Slug:          in.Slug,
		Branch:        branch,
		Base:          base,
		WorktreePath:  worktreePath,
		WorkspacePath: workspacePath,
	}, nil
}

func render(name, tpl string, data map[string]string) (string, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("parse %s %q: %w", name, tpl, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}
