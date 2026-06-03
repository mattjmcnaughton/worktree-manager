package version

// Version is the current version of worktree-manager.
// Override at build time with:
//
//	go build -ldflags "-X github.com/mattjmcnaughton/worktree-manager/internal/version.Version=x.y.z" ./cmd/worktree-manager
var Version = "dev"
