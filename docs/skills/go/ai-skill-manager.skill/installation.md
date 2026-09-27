# Installation and access

## Prerequisites

- Go 1.26 or newer (`go version`), only to install with `go install` or build from a clone.
- Git on `PATH` for `type: github` sources; if `git clone` fails, `aism` downloads the GitHub archive over HTTPS instead. Git uses the caller's own credentials; `aism` needs no tokens or environment variables.
- Make, only to build from a clone.

## Install

A released binary (no Go needed) — `linux`, `darwin` or `windows`, `amd64`; the `.exe` suffix on Windows. Check it against `ai-skill-manager_<version>_checksums.txt` from the same release:

```sh
v=2.0.0; os=linux   # or darwin
curl -fsSLo aism "https://github.com/InsonusK/go-ai-skill-manage/releases/download/v$v/ai-skill-manager_${v}_${os}_amd64"
chmod +x aism && sudo mv aism /usr/local/bin/
```

With `go install` (the binary is `ai-skill-manager`):

```sh
go install github.com/InsonusK/go-ai-skill-manage/cmd/ai-skill-manager@latest
ln -s "$(go env GOPATH)/bin/ai-skill-manager" "$(go env GOPATH)/bin/aism"   # optional short name
```

From a clone (writes `bin/ai-skill-manager` and its copy `bin/aism`):

```sh
git clone git@github.com:InsonusK/go-ai-skill-manage.git
cd go-ai-skill-manage
make build
export PATH="$PWD/bin:$PATH"
```

`aism` and `ai-skill-manager` are the same program; every command below works with either name.

## Verify

```sh
aism --version
aism --help
```

Expected: a version (e.g. `2.0.0`), then the usage text that starts with `Usage: aism [--debug] [--profile] <command> [options]` and lists the `sync` and `validate` commands.
