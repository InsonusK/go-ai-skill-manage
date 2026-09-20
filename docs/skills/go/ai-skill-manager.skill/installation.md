# Installation and access

Prerequisites: Go 1.26+, Make; Git for clone-based acquisition.
From the repository root:

```sh
make build
./bin/aism --version
./bin/aism --help
```

The binary is also available as `bin/ai-skill-manager`.
A built binary needs no Python or Go runtime installation.
The first build downloads pinned modules from go.mod/go.sum.
Git uses the caller's normal Git credentials; no application-specific
credential variables are required. Public GitHub archive fallback needs HTTP access.
