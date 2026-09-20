SHELL := /bin/bash
.DEFAULT_GOAL := build

VERSION := $(shell cat VERSION)
MODULE := github.com/InsonusK/go-ai-skill-manage
LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION)
# gremlins v0.6.0 emits an invalid Go argument with --test-cpu; omit that flag.
GREMLINS_VERSION := v0.6.0
GREMLINS := $(CURDIR)/bin/gremlins
COVERPKG := $(shell go list ./cmd/... ./internal/... | grep -Ev '/(test|gen)(/|$$)' | paste -sd, -)

.PHONY: build run lint unit-test mutation-test test-report test-and-report

build:
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/ai-skill-manager ./cmd/ai-skill-manager
	cp bin/ai-skill-manager bin/aism

run: build
	./bin/aism $(ARGS)

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
	"$(GREMLINS)" unleash --integration --workers=4 --coverpkg=$(COVERPKG) --exclude-files='tools/.*' --exclude-files='gen/.*' --exclude-files='[.]agents/.*' --exclude-files='[.]claude/.*' --exclude-files='deprecated/.*' "$${diff_args[@]}" \
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
