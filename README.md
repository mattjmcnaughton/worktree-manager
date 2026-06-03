# worktree-manager

A Go CLI for managing Git worktrees

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

## Usage

```sh
worktree-manager --help
worktree-manager example [name]
```

### Environment Variables

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `WORKTREE_MANAGER_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## Development

See [docs/development.md](docs/development.md) for setup instructions and common tasks.

## License

MIT — see [LICENSE](LICENSE).

