package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	Hooks    Hooks    `yaml:"hooks,omitempty"`
	// Repos is only meaningful in the global config file. Keys are absolute,
	// symlink-resolved repo roots; values are layered between the global
	// defaults and the repo-local `.worktree-manager.yml`.
	Repos map[string]Config `yaml:"repos,omitempty"`
}

type Defaults struct {
	Base             string `yaml:"base,omitempty"`
	WorktreeBaseDir  string `yaml:"worktree_base_dir,omitempty"`
	BranchTemplate   string `yaml:"branch_template,omitempty"`
	WorktreeTemplate string `yaml:"worktree_template,omitempty"`
	User             string `yaml:"user,omitempty"`
}

// UnmarshalYAML accepts `base_dir` as an alias for `worktree_base_dir` so
// existing wtp (.wtp.yml) configurations parse without modification.
func (d *Defaults) UnmarshalYAML(node *yaml.Node) error {
	type raw Defaults
	aux := struct {
		*raw    `yaml:",inline"`
		BaseDir string `yaml:"base_dir,omitempty"`
	}{raw: (*raw)(d)}
	if err := node.Decode(&aux); err != nil {
		return err
	}
	if d.WorktreeBaseDir == "" && aux.BaseDir != "" {
		d.WorktreeBaseDir = aux.BaseDir
	}
	return nil
}

type Agentic struct {
	Enabled             bool   `yaml:"enabled,omitempty"`
	WorkspaceDir        string `yaml:"workspace_dir,omitempty"`
	CreateTaskWorkspace bool   `yaml:"create_task_workspace,omitempty"`
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

// Load reads global and repo configuration and merges them into a single
// config. Precedence (low to high): global defaults < global.repos[<repoRoot>]
// < repo (.worktree-manager.yml). Missing files are not an error: each is
// treated as empty. repoRoot, when non-empty, is symlink-resolved before
// looking it up in global.repos so users with symlinked checkouts get one
// stable entry.
func Load(globalPath, repoPath, repoRoot string) (*Config, error) {
	global, err := readFile(globalPath)
	if err != nil {
		return nil, fmt.Errorf("read global config: %w", err)
	}
	repo, err := readFile(repoPath)
	if err != nil {
		return nil, fmt.Errorf("read repo config: %w", err)
	}

	merged := mergeConfigs(global, repo, repoRoot)
	merged.applyDefaults()
	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return merged, nil
}

// LookupRepoOverride returns the per-repo override block from global, if any,
// matching repoRoot. Both repoRoot and the YAML keys are canonicalized
// (absolute + symlink-resolved) before comparison so users get one stable
// match regardless of how the path was written.
func LookupRepoOverride(global *Config, repoRoot string) (Config, bool) {
	if global == nil || len(global.Repos) == 0 || repoRoot == "" {
		return Config{}, false
	}
	target := canonicalRepoKey(repoRoot)
	for raw, override := range global.Repos {
		if canonicalRepoKey(raw) == target {
			return override, true
		}
	}
	return Config{}, false
}

func canonicalRepoKey(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
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

func mergeConfigs(global, repo *Config, repoRoot string) *Config {
	out := *global
	out.Repos = nil

	override, _ := LookupRepoOverride(global, repoRoot)

	if override.Version != "" {
		out.Version = override.Version
	}
	if repo.Version != "" {
		out.Version = repo.Version
	}
	out.Defaults = mergeDefaults(mergeDefaults(global.Defaults, override.Defaults), repo.Defaults)
	out.Agentic = mergeAgentic(mergeAgentic(global.Agentic, override.Agentic), repo.Agentic)
	out.Hooks = mergeHooks(mergeHooks(global.Hooks, override.Hooks), repo.Hooks)

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
