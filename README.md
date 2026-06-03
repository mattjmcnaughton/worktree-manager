# worktree-manager

`worktree-manager` is a Go CLI for managing Git worktrees as task-oriented development environments.

> **Status:** early scaffold. The current binary exposes the root command and an `example` command only. The workflow below is the intended product direction and implementation plan, inspired by the agentic worktree skills in `mattjmcnaughton/skills` and by [`satococoa/wtp`](https://github.com/satococoa/wtp).

## Why

Plain `git worktree` is powerful, but a complete task workflow usually involves more than creating a directory:

- deriving a consistent task slug
- creating a branch with a user-specific prefix
- choosing a predictable worktree path
- optionally creating an agent workspace
- fetching shared context repositories
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
  .agentic/sources/<name>/ shared context repositories
  .worktree-manager/tasks/<slug>.json local task metadata
```

The first-class identifier is the task **slug**, not just the branch name.

Example intended layout:

```text
<repo>/
  .worktree-manager.yml        # tracked repo config

  .worktree-manager/           # local runtime state, ignored
    tasks/
      add-semantic-indexing.json

  .agentic/                    # optional, ignored
    sources/
      skills/
      wtp/

    add-semantic-indexing/     # optional task workspace
      metadata.json
      plan.md
      diary.md
      review.md

  .worktrees/                  # local worktrees, ignored or outside repo
    worktree-manager-add-semantic-indexing/
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
8. Optionally fetch configured context repositories into `.agentic/sources/`.
9. Run `post_create` hooks.
10. Print paths and next steps.

Example output:

```text
Created worktree task

Slug:       add-semantic-indexing
Branch:     mattjmcnaughton/add-semantic-indexing
Worktree:   .worktrees/worktree-manager-add-semantic-indexing
Workspace:  disabled
Context:
  ✓ skills -> .agentic/sources/skills
  ✓ wtp    -> .agentic/sources/wtp
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
cd "$(worktree-manager cd add-semantic-indexing)"
```

These commands are planned. They will resolve worktrees by managed slug and task metadata.

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

When disabled, the worktree can still be managed using local runtime metadata under `.worktree-manager/tasks/`.

## Shared context repositories

Repos can declare context sources that should be cloned or refreshed under `.agentic/sources/`, similar to a local implementation of a `fetch-context` workflow.

```yaml
context:
  enabled: true
  fetch_on_create: true
  sources_dir: ".agentic/sources"
  update_existing: true
  sources:
    - name: "skills"
      repo: "mattjmcnaughton/skills"
      ref: "main"
      depth: 1
      required: true

    - name: "wtp"
      repo: "satococoa/wtp"
      ref: "main"
      depth: 1
      required: false
```

Planned behavior for each source:

- If the directory does not exist, clone it.
- If the directory exists and is a git repo, verify the origin and optionally update it.
- If the directory exists but is not a git repo, fail unless a future force flag is supplied.
- If `required: false`, warn and continue on fetch failure.

Explicit context commands are also planned:

```sh
worktree-manager context list
worktree-manager context status
worktree-manager context sync
worktree-manager context sync skills
```

## Configuration

`worktree-manager` should support both global and per-repo config.

```text
Global:
  ~/.config/worktree-manager/config.yml

Repo:
  <repo>/.worktree-manager.yml
```

Precedence:

```text
CLI flags > repo config > global config > defaults
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

context:
  enabled: true
  fetch_on_create: true
  sources_dir: ".agentic/sources"
  sources:
    - name: "skills"
      repo: "mattjmcnaughton/skills"
    - name: "wtp"
      repo: "satococoa/wtp"

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

    - type: symlink
      from: ".agentic/sources"
      to: ".agentic/sources"
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

Source lists should merge by `name`, allowing repo config to override or disable globally configured sources.

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

The current scaffold supports help, version, completion generation, and an example command:

```sh
worktree-manager --help
worktree-manager example [name]
```

Current environment variables:

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `WORKTREE_MANAGER_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Installation

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
