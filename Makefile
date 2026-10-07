SHELL := /bin/bash
.DEFAULT_GOAL := build

# gremlins v0.6.0 emits an invalid Go argument with --test-cpu; omit that flag.
GREMLINS_VERSION := v0.6.0
GREMLINS := $(CURDIR)/bin/gremlins
# Address-space cap (KB) for gremlins and the tests it starts. A mutant can turn
# a loop into an endless one that keeps appending (inline_code.go:22 `!=` -> `==`
# takes ~1.3 GB/s): without a cap it exhausts the machine before gremlins' own
# timeout fires - a GitHub runner is then shut down mid-job. With the cap the
# test dies with "out of memory" and the mutant counts as killed. 2 GB is too
# little for the suite itself (git, mcpserver tests).
MUTATION_MEMORY_LIMIT_KB := 4194304
COVERPKG := $(shell go list ./cmd/... ./internal/... | grep -Ev '/(test|gen)(/|$$)' | paste -sd, -)

.PHONY: build install run profile conformance conformance-python conformance-compare lint unit-test mutation-test test-report test-and-report

build:
	@mkdir -p bin
	go build -trimpath -o bin/ai-skill-manager ./cmd/ai-skill-manager
	cp bin/ai-skill-manager bin/aism

# Put both names into Go's bin folder (GOBIN, else GOPATH/bin), which is in
# PATH: `aism` then runs this build from any folder.
INSTALL_DIR := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

# A running MCP server (`aism mcp` started by an editor) is stopped first:
# writing into a binary that is running fails with "Text file busy". The
# brackets keep pkill from matching the shell that runs this recipe. The old
# files are removed in case the server has not exited yet.
install: build
	@mkdir -p "$(INSTALL_DIR)"
	@pkill -f '(ais[m]|ai-skill-manage[r])( .*)? mcp' && echo "stopped running MCP server" || true
	rm -f "$(INSTALL_DIR)/ai-skill-manager" "$(INSTALL_DIR)/aism"
	cp bin/ai-skill-manager bin/aism "$(INSTALL_DIR)/"
	@echo "installed aism and ai-skill-manager into $(INSTALL_DIR)"

run: build
	./bin/aism $(ARGS)

# Shared scenarios (test/conformance) against the Go CLI, the Python CLI
# (.venv/bin/aism), or both side by side.
conformance: build
	AISM_CLI=$(CURDIR)/bin/aism go test -count=1 ./test/conformance/...

conformance-python:
	AISM_CLI=$(CURDIR)/.venv/bin/aism go test -count=1 ./test/conformance/...

conformance-compare:
	test/conformance/compare.sh

# Profiles sync: MODE=local (default, local clone) or MODE=github; see profiling/README.md.
profile:
	$(MAKE) -C profiling profile MODE=$(or $(MODE),local)

lint:
	go vet ./...

unit-test:
	@mkdir -p tmp/result tmp/report/tests tmp/report/coverage
	@set -o pipefail; go test -v -json -coverpkg=$(COVERPKG) -coverprofile=tmp/report/coverage/coverage.out ./... \
		| tee tmp/report/tests/go-test.json | go run ./tools/normalize_unittest
	@if [ "$(WITH_CODE_COVERAGE)" = "true" ]; then \
		go tool cover -html=tmp/report/coverage/coverage.out -o tmp/report/coverage/index.html; \
		pct=$$(go tool cover -func=tmp/report/coverage/coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
		printf '{"linePct":%s}\n' "$$pct" > tmp/result/coverage-test.json; \
		awk -v pct="$$pct" 'BEGIN { if (pct < 80) { print "Coverage below 80%: " pct; exit 1 } }'; \
	fi
	@cat tmp/result/unit-test.json

mutation-test:
	@mkdir -p bin tmp/result tmp/report/mutation
	@test -x "$(GREMLINS)" || GOBIN="$(CURDIR)/bin" go install github.com/go-gremlins/gremlins/cmd/gremlins@$(GREMLINS_VERSION)
	@diff_args=(); \
	if [ "$(ONLY_DELTA)" = "true" ]; then diff_args=(--diff "$(DELTA_BASE)"); fi; \
	ulimit -v $(MUTATION_MEMORY_LIMIT_KB); \
	"$(GREMLINS)" unleash --integration --workers=4 --coverpkg=$(COVERPKG) --exclude-files='tools/.*' --exclude-files='gen/.*' --exclude-files='[.]agents/.*' --exclude-files='[.]claude/.*' "$${diff_args[@]}" \
		--output tmp/report/mutation/gremlins.json .; code=$$?; \
	go run ./tools/normalize_mutation tmp/report/mutation/gremlins.json; normalizer=$$?; \
	if [ "$$code" -ne 0 ]; then exit "$$code"; fi; exit "$$normalizer"

test-report:
	go run ./tools/test_report

test-and-report:
	@unit_status=0; $(MAKE) unit-test WITH_CODE_COVERAGE=true || unit_status=$$?; \
	mut_status=0; if [ "$$unit_status" -eq 0 ]; then $(MAKE) mutation-test ONLY_DELTA=$(ONLY_DELTA) DELTA_BASE=$(DELTA_BASE) || mut_status=$$?; fi; \
	$(MAKE) test-report; report_status=$$?; \
	if [ "$$unit_status" -ne 0 ]; then exit "$$unit_status"; fi; \
	if [ "$$mut_status" -ne 0 ]; then exit "$$mut_status"; fi; exit "$$report_status"

.PHONY: diagrams
diagrams:
	go run ./tools/diagram-renderer docs/features/sync.md
