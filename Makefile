SHELL := sh

## Every target is a command rather than a file it builds, so each is declared
## .PHONY next to itself. Without that, make compares the target name against
## the directory and does nothing when a file of that name exists — and this
## repo is one `mkdir test` or stray `build` artifact away from `make test`
## reporting success without running anything. Declaring them next to the
## target rather than in one list at the top is what keeps a new target from
## being added without one.

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: cover
cover: ## generate code coverage report
	rm -f cover.out
	go test -run='^Test' -coverprofile=cover.out -coverpkg=.
	go tool cover -func=cover.out

.PHONY: version
version: ## print OS, Go, and golangci versions
	@echo $$0
	@uname -a
	@go version
	@golangci-lint --version

.PHONY: lintverify
lintverify: ## verify the golangci config (downloads its schema over the network)
	golangci-lint config verify

.PHONY: fmt
fmt: ## reformat source code
	go mod tidy
	gofmt -w -s *.go

.PHONY: lint
lint: ## lint and verify repo is already formatted
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	golangci-lint run .

.PHONY: vulncheck
vulncheck: ## scan the module and stdlib for known vulnerabilities (downloads govulncheck and its database over the network)
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: build
build: ## build
	go build ./...

.PHONY: test
test: ## test
	go test ./...

## The race detector reports only races it observes, so this is worth running
## because TestConcurrentTemplateExecution renders one template over shared data
## from several goroutines. Without a test that does that, -race is a slower way
## to get the same pass.
.PHONY: race
race: ## test under the race detector
	go test -race -count=1 ./...

## Each Fuzz* function is discovered by name rather than listed here, so a new
## one added to a _test.go file is swept without anyone remembering to add it
## in a second place — the rule the rest of the package follows for anything
## that would otherwise need to be kept in sync by hand.
##
## go test only fuzzes one target per invocation — a -fuzz pattern matching
## more than one is a hard error — hence the loop. -run=^$$ skips the regular
## Test/Example suite on every iteration; make test already covers that.
## FUZZTIME overrides the per-target budget, e.g. FUZZTIME=60s make fuzz.
.PHONY: fuzz
fuzz: ## fuzz every Fuzz target briefly (FUZZTIME=10s default; downloads nothing, but each target seeds from testdata/fuzz)
	@for name in $$(grep -h '^func Fuzz' *_test.go | sed -E 's/^func (Fuzz[A-Za-z0-9_]*).*/\1/'); do \
		echo "--- $$name ---"; \
		go test -run=^$$ -fuzz="^$${name}$$" -fuzztime="$${FUZZTIME:-10s}" . || exit 1; \
	done

.PHONY: env
env: ## mac osx environment
	brew upgrade
	brew install golangci-lint

.PHONY: clean
clean: ## remove any generated files
	rm -f *.out
	rm -rf testdata
	go clean -testcache
