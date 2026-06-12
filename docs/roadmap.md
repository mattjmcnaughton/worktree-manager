# Roadmap

Ideas that are intentionally out of scope today but worth keeping a pointer to.
Promote an item from this file to a real plan when work on it actually starts.

## Shared context repositories

Some tasks benefit from materializing reference material (style guides, internal
SDK docs, sibling repos, design docs) into the worktree before the agent
starts. A `context` config block could declare named sources that get cloned,
checked out, or symlinked into the new worktree as part of `post_create`, so
plans and prompts can reference them by stable relative paths.

Open questions: how sources are resolved (git, local path, URL), where they
land (inside `.agentic/` vs the worktree root), whether they are read-only or
writable, and how stale/forked state is reconciled.

## Additional CLI verbs

The README and command tests reference verbs that are not yet implemented.
None of them are blocking, but they are the next obvious places to extend the
CLI surface once the core lifecycle (`create` / `delete` / `list` / `pwd`) is
stable.

- `status <slug>` — summarize a managed worktree (branch state, uncommitted
  work, unpushed commits, related task metadata).
- `exec <slug> -- <command>` — run a command in the worktree directory without
  forcing the user to `cd` first.

## Shell integration

`wtp`-style shell integration (`shell-init`, prompt-side completion bootstrap,
chpwd-style hooks) is not provided. If users want `cd $(worktree-manager pwd
<slug>)` to be a single keystroke, a shell function emitted via
`worktree-manager shell-init` is the obvious shape.
