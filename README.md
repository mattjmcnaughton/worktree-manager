# worktree-manager

`worktree-manager` is a Go CLI for managing Git worktrees as task-oriented development environments.

> **Status:** in progress. The binary currently implements `create`, `delete`, `list`, `pwd`, `shell`, and `tmux`. Other commands described below (`status`, `exec`, `list --json`) are still planned. The overall direction is inspired by the agentic worktree skills in `mattjmcnaughton/skills` and by [`satococoa/wtp`](https://github.com/satococoa/wtp).

## Why

Plain `git worktree` is powerful, but a complete task workflow usually involves more than creating a directory:

- deriving a consistent task slug
- creating a branch with a user-specific prefix
- choosing a predictable worktree path
- optionally creating an agent workspace
- copying or linking local development files
- running per-repo setup and cleanup hooks
- safely deleting worktrees and branches when the task is done

`worktree-manager` is intended to make those steps programmable and repeatable.

## Core model

A managed task can have several linked resources:

```text
Required:
  Git worktree
  Git branch

Optional:
  .agentic/<slug>/ task workspace
  $XDG_STATE_HOME/worktree-manager/repos/<sha8>/tasks/<slug>.json runtime metadata
```

The first-class identifier is the task **slug**, not just the branch name.

Example intended layout:

```text
<repo>/
  .worktree-manager.yml        # tracked repo config

  .agentic/                    # optional, ignored
    add-semantic-indexing/     # optional task workspace
      metadata.json
      plan.md
      diary.md
      review.md

  .worktrees/                  # local worktrees, ignored or outside repo
    worktree-manager-add-semantic-indexing/

$XDG_STATE_HOME/worktree-manager/
  repos/<sha8>/                # one dir per managed repo (sha8 of resolved root)
    repo.json                  # sidecar pointing back at the repo root
    tasks/
      add-semantic-indexing.json
```

## Planned workflow

### Create a worktree

```sh
worktree-manager create "add semantic indexing"
worktree-manager create AGE-4
worktree-manager create gh#42
worktree-manager create --slug AGE-4-add-semantic-indexing
worktree-manager create --base main --exec "just gate"
```

Planned `create` behavior:

1. Load global config.
2. Load per-repo config.
3. Merge config with CLI flags taking precedence.
4. Resolve slug, branch, base, and worktree path.
5. Run `pre_create` hooks.
6. Create the git worktree and branch.
7. Optionally create `.agentic/<slug>/`.
8. Run `post_create` hooks.
9. Print paths and next steps.

Example output:

```text
Created worktree task

Slug:       add-semantic-indexing
Branch:     mattjmcnaughton/add-semantic-indexing
Worktree:   .worktrees/worktree-manager-add-semantic-indexing
Workspace:  disabled
```

### Delete a worktree

```sh
worktree-manager delete add-semantic-indexing
worktree-manager delete .worktrees/worktree-manager-add-semantic-indexing
worktree-manager delete --with-branch add-semantic-indexing
worktree-manager delete --force --force-branch add-semantic-indexing
```

Planned `delete` behavior:

1. Resolve the target from slug, path, or current worktree.
2. Refuse to delete the main worktree.
3. Check for uncommitted work.
4. Check for unpushed commits.
5. Run `pre_delete` hooks.
6. Remove the worktree.
7. Optionally delete the branch.
8. Run `post_delete` hooks.
9. Keep `.agentic/<slug>/` by default unless explicitly requested.

### Inspect and operate on worktrees

```sh
worktree-manager list
worktree-manager list --json
worktree-manager status add-semantic-indexing
worktree-manager status --current
worktree-manager exec add-semantic-indexing -- just gate
cd "$(worktree-manager pwd add-semantic-indexing)"
worktree-manager shell add-semantic-indexing
worktree-manager tmux add-semantic-indexing
```

`list`, `pwd`, `shell`, and `tmux` are implemented today. `status`, `exec`, and `list --json` are still planned. All commands resolve worktrees by managed slug and task metadata.

## Optional agent workspace

Creating `.agentic/<slug>/` should be configurable, not mandatory.

```yaml
agentic:
  enabled: true
  workspace_dir: ".agentic"
  create_task_workspace: false
```

CLI flags should override config:

```sh
worktree-manager create "add semantic indexing" --agentic
worktree-manager create "add semantic indexing" --no-agentic
```

When enabled, the task workspace is intended to hold agent lifecycle files such as:

```text
.agentic/<slug>/metadata.json
.agentic/<slug>/plan.md
.agentic/<slug>/diary.md
.agentic/<slug>/review.md
```

When disabled, the worktree can still be managed using the runtime metadata stored under `${XDG_STATE_HOME:-~/.local/state}/worktree-manager/repos/<sha8>/tasks/` (outside the repo, so nothing needs to be gitignored).

## Configuration

`worktree-manager` should support both global and per-repo config.

```text
Global:
  ${XDG_CONFIG_HOME:-~/.config}/worktree-manager/config.yml

Repo:
  <repo>/.worktree-manager.yml
```

The global file may also declare a `repos:` map keyed by absolute repo
root; entries are layered between the global defaults and the
repo-local file (see the precedence table below).

Precedence:

```text
CLI flags > repo config (.worktree-manager.yml) > global repos[<root>] > global defaults
```

Example repo config:

```yaml
version: "1.0"

defaults:
  base: "main"
  worktree_base_dir: ".worktrees"
  branch_template: "{{ user }}/{{ slug }}"
  worktree_template: "{{ repo }}-{{ slug }}"

agentic:
  enabled: true
  workspace_dir: ".agentic"
  create_task_workspace: true

hooks:
  pre_create:
    - type: command
      command: "just gate"
      work_dir: "main"

  post_create:
    - type: copy
      from: ".env"
      to: ".env"
      optional: true

    - type: command
      command: "just init-worktree"
      optional: true
      work_dir: "worktree"

  pre_delete:
    - type: command
      command: "git status --short"
      work_dir: "worktree"

  post_delete:
    - type: command
      command: "git worktree prune"
      work_dir: "main"
```

## wtp compatibility

`worktree-manager` is **configuration-compatible** with [`satococoa/wtp`](https://github.com/satococoa/wtp): a valid `.wtp.yml` parses without modification when read as a `worktree-manager` repo config. Compatibility is scoped to the config file format, not the CLI surface.

### What is compatible

- **File schema.** Top-level `version`, `defaults`, and `hooks` keys match.
- **`defaults.base_dir`.** Accepted as an alias for `defaults.worktree_base_dir`. If both are set, the canonical `worktree_base_dir` wins.
- **Hook types.** `copy`, `symlink`, and `command` use the same field names (`from`, `to`, `command`, `env`, `work_dir`) and the same semantics:
  - `copy` and `symlink` resolve `from` relative to the **main** worktree and `to` relative to the **new** worktree; `to` defaults to `from` when omitted.
  - `copy` recurses into directories and preserves intra-tree symlinks.
  - `command` runs through `sh -c` with `env` merged into the process environment.
- **`post_create` phase.** wtp's only hook phase is supported as-is.

### What is extended

`worktree-manager` adds keys that wtp does not recognize. These are no-ops for wtp but active here:

- Additional hook phases: `pre_create`, `pre_delete`, `post_delete`.
- Per-hook `name` (label in logs) and `optional` (failures are warned and skipped).
- `defaults.base`, `defaults.branch_template`, `defaults.worktree_template`, `defaults.user`.
- The `agentic` block for `.agentic/<slug>/` workspace creation.
- Hook environment variables prefixed `WORKTREE_MANAGER_*` (see [Hooks](#hooks)).

### What is not compatible

- **Config file discovery.** `worktree-manager` reads `<repo>/.worktree-manager.yml`, not `.wtp.yml`. To reuse an existing wtp config, rename or symlink it:
  ```sh
  ln -s .wtp.yml .worktree-manager.yml
  ```
- **CLI surface.** wtp is branch-oriented (`wtp add feature/auth`, `wtp remove`, `wtp cd`). `worktree-manager` is slug-oriented (`worktree-manager create "add semantic indexing"`, `worktree-manager delete <slug>`, `worktree-manager pwd <slug>`). Branch and worktree paths are derived from templates rather than from the branch name directly.
- **Shell integration.** `wtp shell-init` / `wtp hook` and lazy completion bootstrapping are wtp-specific and have no equivalent here yet.

### Using an existing `.wtp.yml`

Given a typical wtp config:

```yaml
version: "1.0"
defaults:
  base_dir: "../worktrees"
hooks:
  post_create:
    - type: copy
      from: ".env"
      to: ".env"
    - type: command
      command: "npm install"
```

**As-is, the file is ignored.** `worktree-manager` only reads `<repo>/.worktree-manager.yml`, so a bare `.wtp.yml` has no effect and the CLI falls back to defaults.

**Once symlinked or renamed**, the file loads and `worktree-manager create "add auth"` does roughly this:

1. **Load and merge defaults.** `base_dir: "../worktrees"` is read as `worktree_base_dir` via the alias. Unset keys fall back: `base="main"`, `branch_template="{{ user }}/{{ slug }}"`, `worktree_template="{{ repo }}-{{ slug }}"`. `agentic` is disabled.
2. **Backfill `user`.** Empty `defaults.user` is filled from `$USER` (falling back to `os/user.Current().Username`), slug-normalized (e.g. `Jane Doe` → `jane-doe`).
3. **Derive the slug.** `"add auth"` becomes `add-auth`.
4. **Resolve paths.** With repo `myrepo`:
   - branch: `jane-doe/add-auth` (from the default template, not from your file)
   - worktree: `../worktrees/myrepo-add-auth`
5. **Create.** Runs `pre_create` hooks (none here), then `git worktree add` with the resolved branch and `base="main"`.
6. **Run `post_create` hooks.** Identical semantics to wtp: `copy .env` reads from the main worktree and writes into the new worktree; `npm install` runs via `sh -c` in the new worktree with `WORKTREE_MANAGER_*` env vars added to the parent environment.
7. **Record task metadata** under `${XDG_STATE_HOME:-~/.local/state}/worktree-manager/repos/<sha8>/tasks/add-auth.json` (alongside a `repo.json` sidecar identifying the source repo). Nothing is written inside the repo itself.

#### What differs from wtp's workflow

Even with the file loaded, the surface behavior is slug-driven, not branch-driven:

| Aspect | wtp | `worktree-manager` with the same file |
| --- | --- | --- |
| Invocation | `wtp add feature/auth` (you name the branch) | `worktree-manager create "..."` (slug → branch via template) |
| Branch name | `feature/auth` verbatim | `<user>/<slug>` from `branch_template` |
| Worktree leaf dir | mirrors branch: `feature/auth/` | `<repo>-<slug>` (flat) |
| `post_create` hooks | run identically | run identically |
| `pwd` / `shell` / `tmux` / `delete` target | branch name | slug |

#### Closer parity with one shared file

To get wtp-style paths while keeping the file usable by both tools, override the templates. wtp ignores the unknown keys; `worktree-manager` honors them:

```yaml
version: "1.0"
defaults:
  base_dir: "../worktrees"
  branch_template: "{{ slug }}"       # ignored by wtp, used here
  worktree_template: "{{ slug }}"     # ignored by wtp, used here
hooks:
  post_create:
    - type: copy
      from: ".env"
      to: ".env"
    - type: command
      command: "npm install"
```

With this, the resolved branch is just `add-auth` and the worktree lives at `../worktrees/add-auth`, matching wtp's layout.

## Hooks

Hooks are planned for worktree lifecycle phases:

```yaml
hooks:
  pre_create: []
  post_create: []
  pre_delete: []
  post_delete: []
```

Initial hook types:

```yaml
- type: copy
  from: ".env"
  to: ".env"
  optional: true

- type: symlink
  from: ".cache"
  to: ".cache"
  optional: true

- type: command
  command: "just init-worktree"
  work_dir: "worktree"
  env:
    FOO: "bar"
```

### Path roots for `copy` and `symlink`

`from` and `to` are interpreted relative to two different anchors so the
same hook can be reused across every worktree without rewriting paths:

- **`from` is rooted at the main worktree.** It names the source that
  already exists in your checked-out main branch.
- **`to` is rooted at the new worktree being created (or, for `pre_delete`
  / `post_delete`, the one being torn down).** It names where the file
  or link should appear.

Both fields accept absolute paths verbatim. When `copy`'s `to` is omitted
it defaults to `from`, so a flat `from: .env` is shorthand for "copy
`<main>/.env` to `<new-worktree>/.env`".

`copy` recurses into directories and preserves intra-tree symlinks, so
`from: .claude/` mirrors the entire `.claude/` tree. `symlink` is single-target
and fails fast if the destination already exists; remove or rename the
existing file out of band if you want a different layout.

Quick example:

```yaml
hooks:
  post_create:
    - type: copy
      from: ".env"            # copies <main>/.env -> <new-worktree>/.env
      to: ".env"
    - type: symlink
      from: "node_modules"    # link <main>/node_modules from the new worktree
      to: "node_modules"
```

Planned hook environment variables:

```text
WORKTREE_MANAGER_REPO_ROOT
WORKTREE_MANAGER_MAIN_WORKTREE
WORKTREE_MANAGER_WORKTREE_PATH
WORKTREE_MANAGER_WORKSPACE_PATH
WORKTREE_MANAGER_BRANCH
WORKTREE_MANAGER_SLUG
WORKTREE_MANAGER_BASE
WORKTREE_MANAGER_PHASE
```

## Current usage

The binary currently supports help, version, completion generation, and the following commands:

```sh
worktree-manager --help

worktree-manager create "add semantic indexing"
worktree-manager create --slug AGE-4-add-semantic-indexing
worktree-manager create "add semantic indexing" --base main --agentic

worktree-manager list
worktree-manager pwd add-semantic-indexing
worktree-manager shell add-semantic-indexing
worktree-manager tmux add-semantic-indexing

worktree-manager delete add-semantic-indexing
worktree-manager delete --with-branch add-semantic-indexing
worktree-manager delete --force --force-branch add-semantic-indexing
```

`status`, `exec`, and `list --json` are not yet implemented.

Current environment variables:

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `WORKTREE_MANAGER_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Installation

Homebrew (macOS and Linux):

```sh
brew install mattjmcnaughton/tap/worktree-manager
```

With Go:

```sh
go install github.com/mattjmcnaughton/worktree-manager/cmd/worktree-manager@latest
```

Or build from source:

```sh
git clone <repo>
cd worktree-manager
go mod tidy
just build
```

## Development

Common commands:

```sh
just fmt              # check formatting
just fmt-fix          # apply gofmt
just vet              # run go vet
just test             # run unit tests
just test-integration # run integration tests
just test-all         # run all tests
just build            # build bin/worktree-manager
just gate             # fmt + vet + test
```

See [docs/development.md](docs/development.md) for setup instructions and more details.

## Inspiration

- The agentic worktree lifecycle skills from `mattjmcnaughton/skills`:
  - `create-worktree`
  - `prep`
  - `build`
  - `review`
  - `create-commit`
  - `create-pr`
  - `merge-pr`
  - `delete-worktree`
- [`satococoa/wtp`](https://github.com/satococoa/wtp), especially its worktree path conventions, hooks, `exec`, `cd`, and config-driven setup.

## License

MIT — see [LICENSE](LICENSE).
