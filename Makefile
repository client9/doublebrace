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
	test -z "$$(gofmt -l *.go)"
	golangci-lint run .

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

.PHONY: env
env: ## mac osx environment
	brew upgrade
	brew install golangci-lint

.PHONY: clean
clean: ## remove any generated files
	rm -f *.out
	go clean -testcache
