# Installation and access

## Prerequisites

- Go 1.26 or newer (`go version`).
- Git on `PATH` for `type: github` sources; if `git clone` fails, `aism` downloads the GitHub archive over HTTPS instead. Git uses the caller's own credentials; `aism` needs no tokens or environment variables.
- Make, only to build from a clone.

## Install

With `go install` (the binary is `ai-skill-manager`, its version prints as `dev`):

```sh
go install github.com/InsonusK/go-ai-skill-manage/cmd/ai-skill-manager@latest
ln -s "$(go env GOPATH)/bin/ai-skill-manager" "$(go env GOPATH)/bin/aism"   # optional short name
```

From a clone (writes `bin/ai-skill-manager` and its copy `bin/aism`, version from the `VERSION` file):

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

Expected: a version (`2.0.0`, or `dev` after `go install`), then the usage text that starts with `Usage: aism [--debug] [--profile] <command> [options]` and lists the `sync` and `validate` commands.
