# Run aism on the skills of ai-skill*.yaml: see the problems, profile sync,
# read the report. Run from this folder (make -C profiling <target>) or
# from the repository root (make profile).
#
#   make validate   -- check the skills, print the problem tree
#   make sync       -- check and write them into out/<mode>/ targets
#   make profile    -- cold + warm sync with CPU/heap profiles (run.sh)
#   make report     -- top CPU and heap functions of the last profile
#   make web        -- the CPU profile in the pprof web UI (port 8080)
#   make python     -- the old Python CLI on the same config, dry run
#   make clean      -- remove out/
#
# MODE=local (default) reads a local clone (sources/ai-skills, cloned once);
# MODE=github downloads the repository on every run. RUN=cold|warm picks the
# profile for report/web.

SHELL := /bin/bash

MODE ?= local
RUN ?= cold
ROOT := ..
AISM := $(ROOT)/bin/aism
PYTHON_AISM := $(ROOT)/.venv/bin/aism
CONFIG_local := ai-skill.local.yaml
CONFIG_github := ai-skill.yaml
CONFIG := $(CONFIG_$(MODE))
OUT := out/$(MODE)
REPO := https://github.com/InsonusK/ai-skills.git
BRANCH := master
SOURCES := $(if $(filter local,$(MODE)),sources/ai-skills/.git)

ifeq ($(CONFIG),)
$(error MODE must be local or github, got "$(MODE)")
endif

.PHONY: build validate sync profile report web python clean

build:
	@$(MAKE) --no-print-directory -C $(ROOT) build >/dev/null

sources/ai-skills/.git:
	git clone --quiet --depth 1 --branch $(BRANCH) $(REPO) sources/ai-skills

validate: build $(SOURCES)
	$(AISM) validate --config $(CONFIG)

sync: build $(SOURCES)
	$(AISM) sync --config $(CONFIG)

profile:
	./run.sh $(MODE)

report:
	@test -f $(OUT)/$(RUN).cpu.prof || { echo "no $(OUT)/$(RUN).cpu.prof: run make profile first" >&2; exit 1; }
	@echo "== CPU, $(MODE)/$(RUN)"
	@go tool pprof -top -nodecount=30 $(OUT)/$(RUN).cpu.prof 2>/dev/null
	@echo "== CPU by cumulative time, $(MODE)/$(RUN)"
	@go tool pprof -top -cum -nodecount=30 $(OUT)/$(RUN).cpu.prof 2>/dev/null | tail -n +7
	@echo "== heap in use at the end, $(MODE)/$(RUN)"
	@go tool pprof -top -sample_index=inuse_space -nodecount=15 $(OUT)/$(RUN).mem.prof 2>/dev/null | tail -n +5
	@echo "== memory totals, $(MODE)/$(RUN)"
	@grep -o 'msg=memory.*' $(OUT)/$(RUN).log || true

web:
	go tool pprof -http=:8080 $(OUT)/$(RUN).cpu.prof

python: $(SOURCES)
	@test -x $(PYTHON_AISM) || { echo "no $(PYTHON_AISM): install the Python CLI into $(ROOT)/.venv" >&2; exit 1; }
	time $(PYTHON_AISM) sync --config $(CONFIG) --dry-run

clean:
	rm -rf out
