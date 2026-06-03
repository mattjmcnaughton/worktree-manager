package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	HookTypeCopy    = "copy"
	HookTypeCommand = "command"
	HookTypeSymlink = "symlink"

	WorkDirMain     = "main"
	WorkDirWorktree = "worktree"
)

type Config struct {
	Version  string   `yaml:"version,omitempty"`
	Defaults Defaults `yaml:"defaults,omitempty"`
	Agentic  Agentic  `yaml:"agentic,omitempty"`
	Context  Context  `yaml:"context,omitempty"`
	Hooks    Hooks    `yaml:"hooks,omitempty"`
}

type Defaults struct {
	Base             string `yaml:"base,omitempty"`
	WorktreeBaseDir  string `yaml:"worktree_base_dir,omitempty"`
	BranchTemplate   string `yaml:"branch_template,omitempty"`
	WorktreeTemplate string `yaml:"worktree_template,omitempty"`
	User             string `yaml:"user,omitempty"`
}

type Agentic struct {
	Enabled             bool   `yaml:"enabled,omitempty"`
	WorkspaceDir        string `yaml:"workspace_dir,omitempty"`
	CreateTaskWorkspace bool   `yaml:"create_task_workspace,omitempty"`
}

type Context struct {
	Enabled        bool            `yaml:"enabled,omitempty"`
	FetchOnCreate  bool            `yaml:"fetch_on_create,omitempty"`
	SourcesDir     string          `yaml:"sources_dir,omitempty"`
	UpdateExisting bool            `yaml:"update_existing,omitempty"`
	Sources        []ContextSource `yaml:"sources,omitempty"`
}

type ContextSource struct {
	Name           string `yaml:"name"`
	Repo           string `yaml:"repo,omitempty"`
	Ref            string `yaml:"ref,omitempty"`
	Depth          int    `yaml:"depth,omitempty"`
	Required       bool   `yaml:"required,omitempty"`
	UpdateExisting *bool  `yaml:"update_existing,omitempty"`
	Disabled       bool   `yaml:"disabled,omitempty"`
}

type Hooks struct {
	PreCreate  []Hook `yaml:"pre_create,omitempty"`
	PostCreate []Hook `yaml:"post_create,omitempty"`
	PreDelete  []Hook `yaml:"pre_delete,omitempty"`
	PostDelete []Hook `yaml:"post_delete,omitempty"`
}

type Hook struct {
	Type     string            `yaml:"type"`
	Name     string            `yaml:"name,omitempty"`
	From     string            `yaml:"from,omitempty"`
	To       string            `yaml:"to,omitempty"`
	Command  string            `yaml:"command,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	WorkDir  string            `yaml:"work_dir,omitempty"`
	Optional bool              `yaml:"optional,omitempty"`
}

// Load reads global and repo configuration, merges them (repo wins), applies
// defaults, and validates the result. Missing files are not an error: each
// is treated as empty.
func Load(globalPath, repoPath string) (*Config, error) {
	global, err := readFile(globalPath)
	if err != nil {
		return nil, fmt.Errorf("read global config: %w", err)
	}
	repo, err := readFile(repoPath)
	if err != nil {
		return nil, fmt.Errorf("read repo config: %w", err)
	}

	merged := mergeConfigs(global, repo)
	merged.applyDefaults()
	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return merged, nil
}

func readFile(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

func mergeConfigs(global, repo *Config) *Config {
	out := *global

	if repo.Version != "" {
		out.Version = repo.Version
	}
	out.Defaults = mergeDefaults(global.Defaults, repo.Defaults)
	out.Agentic = mergeAgentic(global.Agentic, repo.Agentic)
	out.Context = mergeContext(global.Context, repo.Context)
	out.Hooks = mergeHooks(global.Hooks, repo.Hooks)

	return &out
}

func mergeDefaults(g, r Defaults) Defaults {
	out := g
	if r.Base != "" {
		out.Base = r.Base
	}
	if r.WorktreeBaseDir != "" {
		out.WorktreeBaseDir = r.WorktreeBaseDir
	}
	if r.BranchTemplate != "" {
		out.BranchTemplate = r.BranchTemplate
	}
	if r.WorktreeTemplate != "" {
		out.WorktreeTemplate = r.WorktreeTemplate
	}
	if r.User != "" {
		out.User = r.User
	}
	return out
}

func mergeAgentic(g, r Agentic) Agentic {
	out := g
	if r.Enabled {
		out.Enabled = true
	}
	if r.WorkspaceDir != "" {
		out.WorkspaceDir = r.WorkspaceDir
	}
	if r.CreateTaskWorkspace {
		out.CreateTaskWorkspace = true
	}
	return out
}

func mergeContext(g, r Context) Context {
	out := g
	if r.Enabled {
		out.Enabled = true
	}
	if r.FetchOnCreate {
		out.FetchOnCreate = true
	}
	if r.SourcesDir != "" {
		out.SourcesDir = r.SourcesDir
	}
	if r.UpdateExisting {
		out.UpdateExisting = true
	}
	out.Sources = mergeContextSources(g.Sources, r.Sources)
	return out
}

func mergeContextSources(global, repo []ContextSource) []ContextSource {
	indexByName := make(map[string]int)
	out := make([]ContextSource, 0, len(global)+len(repo))
	for _, src := range global {
		indexByName[src.Name] = len(out)
		out = append(out, src)
	}
	for _, src := range repo {
		if idx, ok := indexByName[src.Name]; ok {
			out[idx] = mergeContextSource(out[idx], src)
			continue
		}
		indexByName[src.Name] = len(out)
		out = append(out, src)
	}
	return out
}

func mergeContextSource(g, r ContextSource) ContextSource {
	out := g
	out.Name = r.Name
	if r.Repo != "" {
		out.Repo = r.Repo
	}
	if r.Ref != "" {
		out.Ref = r.Ref
	}
	if r.Depth != 0 {
		out.Depth = r.Depth
	}
	if r.Required {
		out.Required = true
	}
	if r.UpdateExisting != nil {
		out.UpdateExisting = r.UpdateExisting
	}
	if r.Disabled {
		out.Disabled = true
	}
	return out
}

func mergeHooks(g, r Hooks) Hooks {
	out := g
	if len(r.PreCreate) > 0 {
		out.PreCreate = r.PreCreate
	}
	if len(r.PostCreate) > 0 {
		out.PostCreate = r.PostCreate
	}
	if len(r.PreDelete) > 0 {
		out.PreDelete = r.PreDelete
	}
	if len(r.PostDelete) > 0 {
		out.PostDelete = r.PostDelete
	}
	return out
}

func (c *Config) applyDefaults() {
	if c.Defaults.Base == "" {
		c.Defaults.Base = "main"
	}
	if c.Defaults.WorktreeBaseDir == "" {
		c.Defaults.WorktreeBaseDir = ".worktrees"
	}
	if c.Defaults.BranchTemplate == "" {
		c.Defaults.BranchTemplate = "{{ user }}/{{ slug }}"
	}
	if c.Defaults.WorktreeTemplate == "" {
		c.Defaults.WorktreeTemplate = "{{ repo }}-{{ slug }}"
	}
	if c.Agentic.WorkspaceDir == "" {
		c.Agentic.WorkspaceDir = ".agentic"
	}
	if c.Context.SourcesDir == "" {
		c.Context.SourcesDir = ".agentic/sources"
	}
}

// Validate enforces per-hook field requirements across every phase.
func (c *Config) Validate() error {
	phases := []struct {
		name  string
		hooks []Hook
	}{
		{"pre_create", c.Hooks.PreCreate},
		{"post_create", c.Hooks.PostCreate},
		{"pre_delete", c.Hooks.PreDelete},
		{"post_delete", c.Hooks.PostDelete},
	}
	for _, phase := range phases {
		for i, h := range phase.hooks {
			if err := h.Validate(); err != nil {
				return fmt.Errorf("%s hook %d: %w", phase.name, i+1, err)
			}
		}
	}
	return nil
}

func (h *Hook) Validate() error {
	switch h.Type {
	case HookTypeCopy:
		if h.From == "" {
			return errors.New("copy hook requires 'from'")
		}
		if h.Command != "" {
			return errors.New("copy hook must not set 'command'")
		}
	case HookTypeCommand:
		if h.Command == "" {
			return errors.New("command hook requires 'command'")
		}
		if h.From != "" || h.To != "" {
			return errors.New("command hook must not set 'from'/'to'")
		}
	case HookTypeSymlink:
		if h.From == "" || h.To == "" {
			return errors.New("symlink hook requires both 'from' and 'to'")
		}
		if h.Command != "" {
			return errors.New("symlink hook must not set 'command'")
		}
	default:
		return fmt.Errorf("invalid hook type %q (want copy|command|symlink)", h.Type)
	}
	return nil
}
